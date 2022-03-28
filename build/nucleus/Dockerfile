FROM golang:latest as builder

# create a working directory
COPY . /nucleus
WORKDIR /nucleus


# Build binary
RUN GOARCH=amd64 GOOS=linux go build -ldflags="-w -s" -o nucleus cmd/nucleus/*.go
# Uncomment only when build is highly stable. Compress binary.
# RUN strip --strip-unneeded ts
# RUN upx ts

# use a minimal alpine image
FROM nikolaik/python-nodejs:latest
# Installing chromium so that all linux libs get automatically installed for running puppeteer tests
RUN apt update && apt install -y vim git zstd chromium

COPY bundle /usr/local/bin/bundle
RUN chmod +x /usr/local/bin/bundle
ENV SMART_BINARY=/usr/local/bin/bundle

# Install Custom Runners
RUN mkdir /custom-runners
RUN mkdir /tmp/custom-runners

WORKDIR /tmp/custom-runners
RUN npm init -y

RUN npm i --global-style --legacy-peer-deps \
    @lambdatest/test-at-scale-jasmine-runner@~0.1.0 \
    @lambdatest/test-at-scale-mocha-runner@~0.1.0 \
    @lambdatest/test-at-scale-jest-runner@~0.1.0
RUN npm i -g nyc@^15.1.0

RUN tar -zcf /custom-runners/custom-runners.tgz node_modules
RUN rm -rf /tmp/custom-runners

RUN useradd -u 8877 -ms /bin/bash nucleus

# install nvm for nucleus user
RUN su - nucleus -c "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | /bin/bash"

WORKDIR /home/nucleus
# copy the binary from builder
COPY --chown=nucleus:nucleus --from=builder /nucleus/nucleus /usr/local/bin/
# run the binary
COPY ./build/nucleus/entrypoint.sh /
ENTRYPOINT  ["/bin/sh", "/entrypoint.sh"]
