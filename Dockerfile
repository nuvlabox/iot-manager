FROM golang as builder

RUN apt update && apt install -y libusb-1.0.0-dev udev

COPY code /opt/

WORKDIR /opt/peripheral-manager-usb

RUN go mod tidy && go build

# ---

FROM ubuntu:18.04

ARG GIT_BRANCH
ARG GIT_COMMIT_ID
ARG GIT_BUILD_TIME
ARG GITHUB_RUN_NUMBER
ARG GITHUB_RUN_ID

LABEL git.branch=${GIT_BRANCH}
LABEL git.commit.id=${GIT_COMMIT_ID}
LABEL git.build.time=${GIT_BUILD_TIME}
LABEL git.run.number=${GITHUB_RUN_NUMBER}
LABEL git.run.id=${TRAVIS_BUILD_WEB_URL}

RUN apt update && apt install -y libusb-1.0.0-dev udev

RUN apt-get clean autoclean \
    && apt-get autoremove --yes \
    && /bin/bash -c "rm -rf /var/lib/{apt,dpkg,cache,log}/"debian:buster-slim

COPY --from=builder /opt/peripheral-manager-usb/peripheral-manager-usb /usr/sbin

ONBUILD RUN ./license.sh

ENTRYPOINT ["peripheral_manager_usb"]