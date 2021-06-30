FROM golang as builder

RUN apt update && apt install -y libusb-1.0.0-dev




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

RUN apt update && apt-get install -y --no-install-recommends \
                    usbutils \
                    curl \
                    udev \
                    jq \
                    inotify-tools \
                    ca-certificates

RUN apt-get clean autoclean \
        && apt-get autoremove --yes \
        && /bin/bash -c "rm -rf /var/lib/{apt,dpkg,cache,log}/"debian:stretch-slim

# Another way to do this (more complex but more powerful as well) is to install systemd
# inside the Docker image and move the /dev mount into a tmpmount inside the container
# See Balena's example: https://github.com/balena-io-library/base-images/blob/master/balena-base-images/armv7hf/debian/stretch/run/entry.sh

COPY code/app.sh code/license.sh LICENSE /opt/nuvlabox/
COPY code/usb_actions /usr/sbin/

RUN chmod +x /usr/sbin/nuvla*

WORKDIR /opt/nuvlabox/

VOLUME /srv/nuvlabox/shared

ONBUILD RUN ./license.sh

ENTRYPOINT ["./app.sh"]