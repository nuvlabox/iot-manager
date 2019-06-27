#!/usr/bin/env bash

set -e

buspath=$1
devnumber=$2

if [[ -z ${buspath} ]] || [[ -z ${devnumber} ]]
then
    echo "ERR: this script needs the usb bus path and device number as inputs"
    exit 126
fi

real_devnumber=$(echo ${devnumber} | sed 's/^0*//')

device="${buspath}${devnumber}"

if [[ -e "${device}" ]]
then
    simple_lsusb=$(lsusb -s ${devnumber})
    detailed_lsusb=$(lsusb -D ${device})
    tree_lsusb=$(lsusb -t)

    identifier=$(echo ${simple_lsusb} | cut -d " " -f6- | awk -F' ' '{print $1}')
    vendor=$(echo ${detailed_lsusb} | grep idVendor | awk '{ for (i=3; i<=NF; i++) print $i }')
    product=$(echo ${detailed_lsusb} | grep idProduct | awk '{ for (i=3; i<=NF; i++) print $i }')

    classes=$(echo ${tree_lsusb} | grep ${real_devnumber} | awk -F'[,=]' '{print $4","}')

fi

interface="USB"
### TODO: this availability check should come from the system-manager
available=True
###