ARG PYTHON_VERSION=3.8
ARG ALPINE_VERSION=3.12
ARG BASE_IMAGE=python:${PYTHON_VERSION}-alpine${ALPINE_VERSION}

FROM golang:alpine${ALPINE_VERSION} as builder

RUN apk update && apk add libusb-dev udev pkgconfig gcc musl-dev

COPY code /opt/

WORKDIR /opt/peripheral-manager-usb

RUN go mod tidy && go build

# ---

FROM ${BASE_IMAGE}

ARG GIT_BRANCH
ARG GIT_COMMIT_ID
ARG GIT_BUILD_TIME
ARG GITHUB_RUN_NUMBER
ARG GITHUB_RUN_ID
ARG PROJECT_URL

LABEL git.branch=${GIT_BRANCH}
LABEL git.commit.id=${GIT_COMMIT_ID}
LABEL git.build.time=${GIT_BUILD_TIME}
LABEL git.run.number=${GITHUB_RUN_NUMBER}
LABEL git.run.id=${GITHUB_RUN_ID}
LABEL org.opencontainers.image.authors="support@sixsq.com"
LABEL org.opencontainers.image.created=${GIT_BUILD_TIME}
LABEL org.opencontainers.image.url=${PROJECT_URL}
LABEL org.opencontainers.image.vendor="SixSq SA"
LABEL org.opencontainers.image.title="NuvlaBox Peripheral Manager USB"
LABEL org.opencontainers.image.description="Finds and identifies USB peripherals connected to the NuvlaBox"

RUN apk update && apk add libusb-dev udev

COPY --from=builder /opt/peripheral-manager-usb/peripheral-manager-usb /usr/sbin

ONBUILD RUN ./license.sh

ENTRYPOINT ["peripheral-manager-usb"]
