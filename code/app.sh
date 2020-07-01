#!/usr/bin/env bash

header_message="NuvlaBox Peripheral Manager service for USB devices
\n\n
This microservice is responsible for the autodiscovery and classification
of USB devices in the NuvlaBox.
\n\n
Whenever a USB peripheral is added or removed from the NuvlaBox, this
microservice will find its properties and report it to Nuvla.
\n\n
Arguments:\n
  No arguments are expected.\n
  This message will be shown whenever -h, --help or help is provided and a
  command to the Docker container.\n
"

set -e

SOME_ARG="$1"

help_info() {
    echo "COMMAND: ${1}. You have asked for help:"
    echo -e ${header_message}
    exit 0
}

check_existing_peripherals() {
    # $1 is the NuvlaBox ID
    # $2 is the NuvlaBox version

    old_peripherals=$(ls "${PERIPHERALS_DIR}" | sort)

    for old in ${old_peripherals}
    do
        interface=$(jq -r 'select(.interface != null) | .interface' "${PERIPHERALS_DIR}/${old}")
        if [[ "${interface}" == "USB" ]]
        then
            peripheral_nuvla_id=$(jq -r 'select(.id != null) | .id' "${PERIPHERALS_DIR}/${old}")
            if [[ -z ${peripheral_nuvla_id} ]]
            then
                echo "WARN: found an old USB peripheral without a Nuvla ID...removing it locally only!"
                rm -f "${PERIPHERALS_DIR}/${old}"
            else
                echo "INFO: deleting old USB peripheral from Nuvla"
                nuvlabox-delete-usb-peripheral --nuvla-id=${peripheral_nuvla_id} --peripheral-file="${old}"
            fi
         fi
    done

    # re-register existing peripherals
    new_peripherals=''
    lsusb | while read discovered_peripheral
    do
        id=$(echo "${discovered_peripheral}" | awk -F' ' '{print $6}')
        busnum=$(echo "${discovered_peripheral}" | awk -F' ' '{print $2}')
        devnum=$(echo "${discovered_peripheral}" | awk -F'[ :]' '{print $4}')
        bus="/dev/bus/usb/${busnum}/"

        if [[ ! -f "${PERIPHERALS_DIR}/${id}" ]] && [[ "${new_peripherals}" != *"${id}"* ]]
        then
            echo "INFO: registering USB peripheral ${id} during startup. Adding it to Nuvla"
            new_peripherals+=" ${id}"
            nuvlabox-add-usb-peripheral ${bus} ${devnum} ${1} ${2} &
        fi
    done
}


if [[ ! -z ${SOME_ARG} ]]
then
    if [[ "${SOME_ARG}" = "-h" ]] || [[ "${SOME_ARG}" = "--help" ]] || [[ "${SOME_ARG}" = "help" ]]
    then
        help_info ${SOME_ARG}
    else
        echo "WARNING: this container does not expect any arguments, thus they'll be ignored"
    fi
fi

# Until we cannot find the .peripherals directory, we wait
export SHARED="/srv/nuvlabox/shared"
export PERIPHERALS_DIR="${SHARED}/.peripherals"
export CONTEXT_FILE="${SHARED}/.context"

timeout 120 bash -c -- "until [[ -d $PERIPHERALS_DIR ]]
do
    echo 'INFO: waiting for '$PERIPHERALS_DIR
    sleep 3
done"

# Finds the context file in the shared volume and extracts the UUID from there
timeout 120 bash -c -- "until [[ -f $CONTEXT_FILE ]]
do
    echo 'INFO: waiting for NuvlaBox activation and context file '$CONTEXT_FILE
    sleep 3
done"

nuvlabox_id=$(jq -r .id ${CONTEXT_FILE})
nuvlabox_version=$(jq -r .version ${CONTEXT_FILE})

echo "INFO: checking for existing peripherals..."
check_existing_peripherals ${nuvlabox_id} ${nuvlabox_version}
echo "INFO: start listening for USB related events in ${nuvlabox_id}..."

# Using inotify instead of udev
# To use udev, please check the Dockerfile for an implementation reference with systemd-udev

pipefail=$(date +%s)

mkfifo ${pipefail}
inotifywait -m -q -r /dev/bus/usb -e CREATE -e DELETE --csv > ${pipefail} &
while read event
do
    echo ${event}
    devnumber=$(echo ${event} | awk -F',' '{print $NF}')
    buspath=$(echo ${event} | awk -F',' '{print $1}')
    action=$(echo ${event} | awk -F',' '{print $2}')

    if [[ "${action}" = "CREATE" ]]
    then
        echo "INFO: creating USB peripheral in Nuvla"
        nuvlabox-add-usb-peripheral ${buspath} ${devnumber} ${nuvlabox_id} ${nuvlabox_version} &
    fi

    if [[ "${action}" = "DELETE" ]]
    then
        echo "INFO: deleting USB peripheral from Nuvla"
        nuvlabox-delete-usb-peripheral --device-path="${buspath}${devnumber}" &
    fi
done < ${pipefail}
