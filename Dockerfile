FROM golang:alpine3.12 as builder

RUN apk update && apk add libusb-dev udev pkgconfig gcc musl-dev

COPY code /opt/

WORKDIR /opt/peripheral-manager-usb

RUN go mod tidy && go build

# ---

FROM alpine:3.12

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

RUN apk update && apk add libusb-dev udev

COPY --from=builder /opt/peripheral-manager-usb/peripheral-manager-usb /usr/sbin

ONBUILD RUN ./license.sh

ENTRYPOINT ["peripheral_manager_usb"]