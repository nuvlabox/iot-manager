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

if [[ ! -z ${SOME_ARG} ]]
then
    if [[ "${SOME_ARG}" = "-h" ]] || [[ "${SOME_ARG}" = "--help" ]] || [[ "${SOME_ARG}" = "help" ]]
    then
        help_info ${SOME_ARG}
    else
        echo "WARNING: this container does not expect any arguments, thus they'll be ignored"
    fi
fi

echo "INFO: start listening for USB related events..."

# Using inotify instead of udev
# To use udev, please check the Dockerfile for an implementation reference with systemd-udev

pipefail=$(date +%s)

mkfifo ${pipefail}
inotifywait -m -q -r /dev/bus/usb -e CREATE -e DELETE --csv > ${pipefail}
while read event
do
    echo ${event}
    devnumber=$(echo ${event} | awk -F',' '{print $NF}')
    buspath=$(echo ${event} | awk -F',' '{print $1}')
    action=$(echo ${event} | awk -F',' '{print $2}')

    if [[ "${action}" = "CREATE" ]]
    then
        echo "INFO: creating USB peripheral in Nuvla"
        nuvlabox-add-usb-peripheral ${buspath} ${devnumber}
    fi

    if [[ "${action}" = "DELETE" ]]
    then
        echo "INFO: deleting USB peripheral from Nuvla"
        nuvlabox-delete-usb-peripheral ${buspath} ${devnumber}
    fi
done < ${pipefail}
