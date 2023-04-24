package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	log "github.com/sirupsen/logrus"
)

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

var (
	KUBERNETES_SERVICE_HOST, k8s_ok = os.LookupEnv("KUBERNETES_SERVICE_HOST")
	namespace                       = getEnv("MY_NAMESPACE", "nuvlaedge")
)

var lsUsbFunctional bool = false

func getSerialNumberForDevice(devicePath string) string {
	cmd := exec.Command("udevadm", "info", "--attribute-walk", devicePath)

	stdout, cmdErr := cmd.Output()
	var serialNumber string = ""
	var backupSerialNumber string = ""

	if cmdErr != nil {
		log.Errorf("Unable to run udevadm for device %s. Reason: %s", devicePath, cmdErr.Error())
		return serialNumber
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		if strings.Contains(line, "serial") {
			if strings.Contains(line, ".usb") {
				backupSerialNumber = strings.Split(line, "\"")[1]
				continue
			}
			serialNumber = strings.Split(line, "\"")[1]
			break
		}
	}

	if len(serialNumber) == 0 && len(backupSerialNumber) > 0 {
		serialNumber = backupSerialNumber
	}

	return serialNumber
}

func onContextError() {
	if !lsUsbFunctional {
		log.Warn("Unable to initialize USB discovery. Host might be incompatible with this peripheral manager. Trying again later...")
		time.Sleep(10 * time.Second)
		log.Info(string(debug.Stack()))
		os.Exit(0)
	}
}

func getUsbContext() *gousb.Context {
	defer onContextError()
	c := gousb.NewContext()
	lsUsbFunctional = true
	return c
}

func main() {
	log.Info("Peripheral Manager USB has started")

	// Only one context should be needed for an application.  It should always be closed.
	ctx := getUsbContext()
	defer ctx.Close()

	var available bool = true
	var devInterface string = "USB"
	var videoFilesBasedir string = "/dev/"

	for true {
		// Default name for USB
		name := "UNNAMED USB Device"

		_, devErr := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			identifier := fmt.Sprintf("%s:%s", desc.Vendor, desc.Product)

			devicePath := fmt.Sprintf("/dev/bus/usb/%03d/%03d", desc.Bus, desc.Address)

			vendor := usbid.Vendors[desc.Vendor]

			product := vendor.Product[desc.Product]

			description := fmt.Sprintf("%s device [%s] with ID %s. Protocol: %s",
				devInterface,
				product,
				identifier,
				usbid.Classify(desc))

			if product != nil {
				name = fmt.Sprintf("%s", product)
			} else {
				name = fmt.Sprintf("%s with ID %s", name, identifier)
			}

			classesAux := make(map[string]bool)

			classes := make([]interface{}, 0)

			for _, cfg := range desc.Configs {
				for _, intf := range cfg.Interfaces {
					for _, ifSetting := range intf.AltSettings {
						class := fmt.Sprintf("%s", usbid.Classes[ifSetting.Class])
						if _, exists := classesAux[class]; !exists {
							classesAux[class] = true
							classes = append(classes, class)
						}
					}
				}
			}

			serialNumber := getSerialNumberForDevice(devicePath)

			peripheral := map[string]interface{}{
				"name":        name,
				"description": description,
				"interface":   devInterface,
				"identifier":  identifier,
				"classes":     classes,
				"available":   available,
				//"resources": n/a
				// Leaving out the resources attribute since this is only used for
				// block devices, which at the moment are already monitored by the
				// NB Agent, so no need to duplicate the same information.
				// To re-implement this attribute, check the raw legacy code in [1]
			}

			if len(vendor.Name) > 0 {
				peripheral["vendor"] = vendor.Name
			}

			if product != nil {
				peripheral["product"] = fmt.Sprintf("%s", product)
			}

			if len(devicePath) > 0 {
				peripheral["device-path"] = devicePath
			}

			if len(serialNumber) > 0 {
				peripheral["serial-number"] = serialNumber
			}

			devFiles, vfErr := ioutil.ReadDir(videoFilesBasedir)
			if vfErr != nil {
				log.Errorf("Unable to read files under %s. Reason: %s", videoFilesBasedir, vfErr.Error())
				return false
			}

			for _, df := range devFiles {
				if strings.HasPrefix(df.Name(), "video") {
					vfSerialNumber := getSerialNumberForDevice(videoFilesBasedir + df.Name())
					if vfSerialNumber == serialNumber {
						peripheral["video-device"] = videoFilesBasedir + df.Name()
						break
					}
				}
			}

			// we now have a peripheral categorized, but is it new
			//peripheralBody, _ := json.Marshal(peripheral)

			log.Info("Usb found with feats: %s", peripheral)
			return false
		})

		if devErr != nil {
			log.Errorf("A problem occurred while listing the USB peripherals %s. Continuing...", devErr)
		}

		time.Sleep(30 * time.Second)
	}
}
