FROM python:3-alpine AS libsensors-builder

RUN apk add --no-cache gcc=8.3.0-r0 linux-headers=4.18.13-r1 musl-dev=1.1.20-r4 make=4.2.1-r2 git=2.20.1-r0 \
        flex=2.6.4-r1 bison=3.0.5-r0

RUN git clone https://github.com/lm-sensors/lm-sensors.git /tmp/lm-sensors

WORKDIR /tmp/lm-sensors

RUN make install

# ---

FROM python:3-alpine

ARG GIT_BRANCH
ARG GIT_COMMIT_ID
ARG GIT_DIRTY
ARG GIT_BUILD_TIME
ARG TRAVIS_BUILD_NUMBER
ARG TRAVIS_BUILD_WEB_URL

LABEL git.branch=${GIT_BRANCH}
LABEL git.commit.id=${GIT_COMMIT_ID}
LABEL git.dirty=${GIT_DIRTY}
LABEL git.build.time=${GIT_BUILD_TIME}
LABEL travis.build.number=${TRAVIS_BUILD_NUMBER}
LABEL travis.build.web.url=${TRAVIS_BUILD_WEB_URL}

COPY --from=libsensors-builder /usr/local/bin/sensors /usr/local/bin/sensors
COPY --from=libsensors-builder /usr/local/sbin/sensors-detect /usr/local/sbin/sensors-detect
COPY --from=libsensors-builder /usr/local/lib/libsensors.so.5 /usr/local/lib/libsensors.so.5

COPY code/ /opt/nuvlabox/

WORKDIR /opt/nuvlabox/

RUN apk add --no-cache perl=5.26.3-r0


