package peripheral_manager_usb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	log "github.com/sirupsen/logrus"
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

func get_serial_number_for_device(device_path string) (string, bool) {
	cmd := exec.Command("udevadm", "info", "--attribute-walk", device_path)

	stdout, cmd_err := cmd.Output()
	var serial_number string = ""

	if cmd_err != nil {
		log.Errorf("Unable to run udevadm for device %s. Reason: %s", device_path, cmd_err.Error())
		return serial_number, false
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		if strings.Contains(line, "serial") && !strings.Contains(line, ".usb") {
			serial_number = strings.Split(line, "\"")[1]
			break
		}
	}

	if len(serial_number) > 0 {
		return serial_number, true
	} else {
		return serial_number, false
	}

}

func make_agent_request(method string, url string, json_body []byte) (bool, error) {
	agent_client := &http.Client{}

	var req *http.Request
	var set_error error
	if len(json_body) > 0 {
		req, set_error = http.NewRequest(method, url, bytes.NewBuffer(json_body))
		if set_error != nil {
			log.Fatal(set_error)
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	} else {
		req, set_error = http.NewRequest(method, url, bytes.NewBuffer(json_body))
		if set_error != nil {
			log.Fatal(set_error)
		}
	}

	_, req_error := agent_client.Do(req)
	if req_error != nil {
		return false, req_error
	}

	return true, nil
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

	var agent_api_peripherals string = agent_api_base_url + "/peripheral"

	var agent_api_get_usb_devices string = agent_api_peripherals + "?parameter=interface&value=USB"

	// Only one context should be needed for an application.  It should always be closed.
	ctx := gousb.NewContext()
	defer ctx.Close()

	var available bool = true
	var dev_interface string = "USB"
	var video_files_basedir string = "/dev/"

	for true {
		existing_devices := make(map[string]map[string]interface{})
		err := json.Unmarshal([]byte(get_existing_devices(agent_api_get_usb_devices)), &existing_devices)

		if err != nil {
			log.Fatalf("Cannot infer if there are already other USB devices registered in the NuvlaBox. Will not continue. Error: %s", err)
		}

		name := "UNNAMED USB Device"

		_, dev_err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			identifier := fmt.Sprintf("%s:%s", desc.Vendor, desc.Product)

			device_path := fmt.Sprintf("/dev/bus/usb/%03d/%03d", desc.Bus, desc.Address)

			vendor := usbid.Vendors[desc.Vendor]

			product := vendor.Product[desc.Product]

			description := fmt.Sprintf("%s device [%s] with ID %s. Protocol: %s",
				dev_interface,
				product.Name,
				identifier,
				usbid.Classify(desc))

			if product != nil {
				name = fmt.Sprintf("%s", product)
			} else {
				name = description
			}

			classes_aux := make(map[string]bool)

			classes := make([]interface{}, 0)

			for _, cfg := range desc.Configs {
				for _, intf := range cfg.Interfaces {
					for _, ifSetting := range intf.AltSettings {
						class := fmt.Sprintf("%s", usbid.Classes[ifSetting.Class])
						if _, exists := classes_aux[class]; !exists {
							classes_aux[class] = true
							classes = append(classes, class)
						}
					}
				}
			}

			serial_number, ok := get_serial_number_for_device(device_path)

			if !ok {
				return false
			}

			peripheral := map[string]interface{}{
				"name":        name,
				"description": description,
				"interface":   dev_interface,
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

			if len(product.Name) > 0 {
				peripheral["product"] = product.Name
			}

			if len(device_path) > 0 {
				peripheral["device-path"] = device_path
			}

			if len(serial_number) > 0 {
				peripheral["serial-number"] = serial_number
			}

			dev_files, vf_err := ioutil.ReadDir(video_files_basedir)
			if vf_err != nil {
				log.Errorf("Unable to read files under %s. Reason: %s", video_files_basedir, vf_err.Error())
				return false
			}

			for _, df := range dev_files {
				if strings.HasPrefix(df.Name(), "video") {
					vf_serial_number, _ := get_serial_number_for_device(video_files_basedir + df.Name())
					if vf_serial_number == serial_number {
						peripheral["video-device"] = video_files_basedir + df.Name()
						break
					}
				}
			}

			// we now have a peripheral categorized, but is it new?
			old_peripheral, is_old := existing_devices[identifier]

			peripheral_body, _ := json.Marshal(peripheral)

			if !is_old {
				// this peripheral didn't exist before, so let's register (POST) it
				log.Infof("Registering new peripheral %s with identifier %s", name, identifier)
				ok, req_err := make_agent_request("POST", agent_api_peripherals, peripheral_body)

				if !ok {
					log.Errorf("Unable to register new peripheral %s (%s). Reason: %s", name, identifier, req_err)
				}
			} else {
				// peripheral already registered
				// the NB adds an ID, parent and Version to each peripheral...so let's remove those to simplify
				delete(old_peripheral, "id")
				delete(old_peripheral, "version")
				delete(old_peripheral, "parent")
				if reflect.DeepEqual(old_peripheral, peripheral) {
					// this peripheral was already registered, and it has not changed since then
					// nothing to do
					delete(existing_devices, identifier)
				} else {
					// the peripheral already exists, but apparently it has changed
					// need to update (PUT) it
					log.Infof("An existing peripheral (%s - %s) has changed. Updating it", identifier, name)
					ok, req_err := make_agent_request("PUT", agent_api_peripherals+"/"+identifier, peripheral_body)

					if !ok {
						log.Errorf("Unable to update peripheral %s (%s). Reason: %s", name, identifier, req_err)
					}
					delete(existing_devices, identifier)
				}
			}

			return false
		})

		// whatever devices are left in existing_peripherals, have not in the system anymore
		// so let's DELETE them
		if len(existing_devices) > 0 {
			for del_peripheral_id, _ := range existing_devices {
				log.Infof("Peripheral %s (%s) is no longer visible. Removing it", del_peripheral_id, name)
				ok, req_err := make_agent_request("DELETE", agent_api_peripherals+"/"+del_peripheral_id, []byte(""))

				if !ok {
					log.Errorf("Unable to delete old peripheral %s. Reason: %s", del_peripheral_id, req_err)
				}
			}
		}

		if dev_err != nil {
			log.Errorf("A problem occurred while listing the USB peripherals %s. Continuing...", dev_err)
		}

		time.Sleep(30 * time.Second)
	}
}

// [1]
/*
resources=''
if [[ -n "$device_serial" ]]
then
  matching_usb_disks=$(ls -d ${disk_by_id}/* | grep usb | grep "${device_serial}" || echo '')
  for disk in ${matching_usb_disks}
  do
    block_device=$(readlink -f ${disk})
    device_name=$(echo $block_device | awk -F'/' '{print $NF}')

    partitions=$(lsblk $block_device -o NAME,MOUNTPOINT,FSUSE%,SIZE -f -i -n -P)
    export $(echo "${partitions}" | grep "\"$device_name\"" | tr -d '%' | tr -d '"')

    capacity_gb=$SIZE
    unit=$block_device

    resource_json="{\"unit\": \"${unit}\", \"capacity\": \"${capacity_gb}\"}"

    # this section is commented because load changes with time, and at the moment this peripheral manager is
    # opportunistic. It doesn't monitor peripherals periodically.
    # If that changes, then uncomment this sections
  #  if [ -n "$MOUNTPOINT" ] && [ -n "FSUSE" ]
  #  then
  #    # disk is mounted so we can get its usage
  ##    load=$(df -h $block_device --output=pcent | tail -1 | tr -d ' ' | tr -d '%')
  #    load=$FSUSE
  #    resource_json="${resource_json},\"load\": ${load}}"
  #  else
  #    resource_json="${resource_json}}"
  #  fi

    resources="${resources}${resource_json},"
  done
  resources=$(echo "${resources}" | sed 's/\(.*\),/\1 /')
fi
*/
