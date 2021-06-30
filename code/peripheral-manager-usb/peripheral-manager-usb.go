package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	log "github.com/sirupsen/logrus"
	// "github.com/google/gousb/usbid"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

var (
	KUBERNETES_SERVICE_HOST, k8s_ok = os.LookupEnv("KUBERNETES_SERVICE_HOST")
	namespace                       = getenv("MY_NAMESPACE", "nuvlabox")
)

func wait_for_nuvlabox_bootstrap(healthcheck_endpoint string) bool {
	log.Info("Waiting for NuvlaBox to finish bootstrapping (looking at " + healthcheck_endpoint + ")")
	defer log.Info("Agent API is ready")
	for true {
		resp, _ := http.Get(healthcheck_endpoint)

		if resp != nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			break
		}
		time.Sleep(3 * time.Second)
	}

	return true
}

func get_existing_devices(query_url string) string {
	log.Info("Getting existing USB peripherals through ", query_url)

	resp, err := http.Get(query_url)
	if err != nil {
		log.Fatalf("Unable to retrieve existing USB devices via %s. Error: %s", query_url, err)
	}
	body, _ := io.ReadAll(resp.Body)

	return string(body)
}

func main() {
	log.Info("Peripheral Manager USB has started")

	var agent_dns_name string
	if k8s_ok {
		agent_dns_name = "agent." + namespace
	} else {
		agent_dns_name = "localhost:5080"
	}

	var agent_api_base_url string = "http://" + agent_dns_name + "/api"

	wait_for_nuvlabox_bootstrap(agent_api_base_url + "/healthcheck")

	var agent_api_get_usb_devices string = agent_api_base_url + "/peripheral?parameter=interface&value=USB"

	existing_devices := make(map[string]interface{})
	err := json.Unmarshal([]byte(get_existing_devices(agent_api_get_usb_devices)), &existing_devices)

	if err != nil {
		log.Fatalf("Cannot infer if there are already other USB devices registered in the NuvlaBox. Will not continue. Error: %s", err)
	}
	// END OF check_existing_peripherals
	//

	// Only one context should be needed for an application.  It should always be closed.
	ctx := gousb.NewContext()
	defer ctx.Close()

	var available bool = true
	var dev_interface string = "USB"
	var name string
	var description string
	var identifier string

	for true {
		name = "UNNAMED USB Device"
		identifier, description = "", ""

		_, dev_err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			identifier = fmt.Sprintf("%s:%s", desc.Vendor, desc.Product)
			log.Info(identifier)

			product := usbid.Vendors[desc.Product]
			log.Info(product)

			vendor := usbid.Vendors[desc.Vendor]
			log.Info(vendor)

			description = fmt.Sprintf("%s %s %s", dev_interface, identifier, product)
			log.Info(description)

			if product != nil {
				name = fmt.Sprintf("%s", product)
			} else {
				name = description
			}
			log.Info(name)

			fmt.Printf("  Protocol: %s\n", usbid.Classify(desc))

			// The configurations can be examined from the DeviceDesc, though they can only
			// be set once the device is opened.  All configuration references must be closed,
			// to free up the memory in libusb.
			for _, cfg := range desc.Configs {
				// This loop just uses more of the built-in and usbid pretty printing to list
				// the USB devices.
				fmt.Printf("  %s:\n", cfg)
				for _, intf := range cfg.Interfaces {
					fmt.Printf("    --------------\n")
					for _, ifSetting := range intf.AltSettings {
						fmt.Printf("    %s\n", ifSetting)
						fmt.Printf("      %s\n", usbid.Classify(ifSetting))
						for _, end := range ifSetting.Endpoints {
							fmt.Printf("      %s\n", end)
						}
					}
				}
				fmt.Printf("    --------------\n")
			}

			// After inspecting the descriptor, return true or false depending on whether
			// the device is "interesting" or not.  Any descriptor for which true is returned
			// opens a Device which is retuned in a slice (and must be subsequently closed).
			return false
		})

		if dev_err != nil {
			log.Errorf("A problem occurred while listing the USB peripherals %s. Continuing...", dev_err)
		}

		time.Sleep(30 * time.Second)
	}
}
