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
FROM nikolaik/python-nodejs:python3.10-nodejs16-slim

ARG VERSION
ENV VERSION=$VERSION

# Installing chromium so that all linux libs get automatically installed for running puppeteer tests
RUN apt update && apt install -y git zstd chromium curl unzip zip xmlstarlet build-essential
RUN curl -LJO https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.18.3.linux-amd64.tar.gz

COPY bundle /usr/local/bin/bundle
RUN chmod +x /usr/local/bin/bundle
ENV SMART_BINARY=/usr/local/bin/bundle

# Install Custom Runners
RUN mkdir /custom-runners
RUN mkdir /tmp/custom-runners

WORKDIR /tmp/custom-runners
RUN npm init -y
RUN npm install -g pnpm
RUN npm i --global-style --legacy-peer-deps \
    @lambdatest/test-at-scale-jasmine-runner@~0.3.0 \
    @lambdatest/test-at-scale-mocha-runner@~0.3.0 \
    @lambdatest/test-at-scale-jest-runner@~0.3.0
RUN npm i -g nyc@^15.1.0

RUN tar -zcf /custom-runners/custom-runners.tgz node_modules
RUN rm -rf /tmp/custom-runners
RUN mkdir /home/nucleus
RUN mkdir /home/nucleus/.nvm
ENV NVM_DIR=/home/nucleus/.nvm

ENV GOROOT /usr/local/go
ENV GOPATH /home/nucleus
ENV PATH /usr/local/go/bin:/home/nucleus/bin:$PATH

COPY ./build/nucleus/golang/server /home/nucleus

RUN chmod 744 /home/nucleus/server

# install nvm for nucleus user
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | /bin/bash

WORKDIR /home/nucleus
# copy the binary from builder
COPY --from=builder /nucleus/nucleus /usr/local/bin/
# run the binary
COPY ./build/nucleus/entrypoint.sh /

RUN apt update -y && apt upgrade -y

RUN  curl -s https://get.sdkman.io | bash
RUN /bin/bash -c "source $HOME/.sdkman/bin/sdkman-init.sh;sdk install java 18.0.1-oracle"

ENV JAVA_HOME="/root/.sdkman/candidates/java/current"
ENV PATH=$JAVA_HOME:$PATH
ENV PATH=$JAVA_HOME/bin:$PATH

ARG MAVEN_VERSION=3.6.3

# Define a constant with the working directory
ARG USER_HOME_DIR="/root"
# Define the URL where maven can be downloaded from
ARG BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries

# Create the directories, download maven, validate the download, install it, remove downloaded file and set links
RUN mkdir -p /usr/share/maven /usr/share/maven/ref \
  && echo "Downlaoding maven" \
  && curl -fsSL -o /tmp/apache-maven.tar.gz ${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
  \
  && echo "Unziping maven" \
  && tar -xzf /tmp/apache-maven.tar.gz -C /usr/share/maven --strip-components=1 \
  \
  && echo "Cleaning and setting links" \
  && rm -f /tmp/apache-maven.tar.gz \
  && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn

# Define environmental variables required by Maven, like Maven_Home directory and where the maven repo is located
ENV MAVEN_HOME /usr/share/maven
RUN mkdir -p /home/nucleus/.m2

#update settings.xml file for new maven local repo location
RUN xmlstarlet ed -O --inplace -N a='http://maven.apache.org/SETTINGS/1.0.0' -s /a:settings --type elem --name "localRepository" -v /home/nucleus/.m2/repository /usr/share/maven/conf/settings.xml

COPY ./build/nucleus/java/test-at-scale-java.jar /
RUN curl -o /home/nucleus/junit-platform-console-standalone-1.8.2.jar https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/1.8.2/junit-platform-console-standalone-1.8.2.jar
COPY ./build/nucleus/entrypoint.sh /
ENTRYPOINT  ["/bin/sh", "/entrypoint.sh"]