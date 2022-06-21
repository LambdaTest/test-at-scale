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
# Installing chromium so that all linux libs get automatically installed for running puppeteer tests
RUN apt update && apt install -y git zstd chromium curl unzip zip

# Installing chromium so that all linux libs get automatically installed for running puppeteer tests
COPY bundle /usr/local/bin/bundle
RUN chmod +x /usr/local/bin/bundle
ENV SMART_BINARY=/usr/local/bin/bundle

# Install Custom Runners
RUN mkdir /custom-runners
RUN mkdir /tmp/custom-runners

WORKDIR /tmp/custom-runners
RUN npm init -y

RUN npm i --global-style --legacy-peer-deps \
    @lambdatest/test-at-scale-jasmine-runner@~0.2.0 \
    @lambdatest/test-at-scale-mocha-runner@~0.2.0 \
    @lambdatest/test-at-scale-jest-runner@~0.2.0
RUN npm i -g nyc@^15.1.0

RUN tar -zcf /custom-runners/custom-runners.tgz node_modules
RUN rm -rf /tmp/custom-runners
RUN useradd -u 8877 -ms /bin/bash nucleus

RUN curl -LJO https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.18.3.linux-amd64.tar.gz
RUN apt update && apt install -y build-essential
ENV PATH /usr/local/go/bin:$PATH
RUN go version

ENV GOROOT /usr/local/go
ENV GOPATH /home/nucleus
ENV PATH /home/nucleus/bin:$PATH

COPY ./build/nucleus/golang/server /home/nucleus

RUN chown nucleus:nucleus /home/nucleus && \
    chmod 744 /home/nucleus

# install nvm for nucleus user
RUN su - nucleus -c "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | /bin/bash"

WORKDIR /home/nucleus
# copy the binary from builder
COPY --chown=nucleus:nucleus --from=builder /nucleus/nucleus /usr/local/bin/
# run the binary
COPY ./build/nucleus/entrypoint.sh /

#########################################
# FROM openjdk:11-jdk
RUN apt update -y && apt upgrade -y


# RUN apt-get install openjdk-18-jdk -y


#RUN export PATH=$PATH:/usr/local/openjdk-11/bin

USER nucleus
RUN  curl -s https://get.sdkman.io | bash
RUN /bin/bash -c "source $HOME/.sdkman/bin/sdkman-init.sh;sdk install java 11.0.12-open"

ENV JAVA_HOME="/home/nucleus/.sdkman/candidates/java/current"
ENV PATH=$JAVA_HOME:$PATH
ENV PATH=$JAVA_HOME/bin:$PATH
USER root
RUN apt-get install -y wget

# RUN wget https://download.oracle.com/java/18/latest/jdk-18_linux-x64_bin.deb
# RUN apt install ./jdk-18_linux-x64_bin.deb -y
# ENV JAVA_HOME="/usr/lib/jvm/jdk-18"
# ENV PATH $JAVA_HOME:$PATH
# ENV PATH $JAVA_HOME/bin:$PATH

ARG MAVEN_VERSION=3.6.3

# 2- Define a constant with the working directory
ARG USER_HOME_DIR="/root"
# # 3- Define the SHA key to validate the maven download
# ARG SHA=b4880fb7a3d81edd190a029440cdf17f308621af68475a4fe976296e71ff4a4b546dd6d8a58aaafba334d309cc11e638c52808a4b0e818fc0fd544226d952544

# 4- Define the URL where maven can be downloaded from
ARG BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries

# 5- Create the directories, download maven, validate the download, install it, remove downloaded file and set links
RUN mkdir -p /usr/share/maven /usr/share/maven/ref \
  && echo "Downlaoding maven" \
  && curl -fsSL -o /tmp/apache-maven.tar.gz ${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
  \
#   && echo "Checking download hash" \
#   && echo "${SHA}  /tmp/apache-maven.tar.gz" | sha512sum -c - \
#   \
  && echo "Unziping maven" \
  && tar -xzf /tmp/apache-maven.tar.gz -C /usr/share/maven --strip-components=1 \
  \
  && echo "Cleaning and setting links" \
  && rm -f /tmp/apache-maven.tar.gz \
  && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn

# 6- Define environmental variables required by Maven, like Maven_Home directory and where the maven repo is located
ENV MAVEN_HOME /usr/share/maven
ENV MAVEN_CONFIG "/home/nucleus/.m2"

##########################################

# run the binary
COPY ./build/nucleus/test-at-scale-java-1.0-jar-with-dependencies.jar /
RUN su - nucleus -c "wget https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/1.8.2/junit-platform-console-standalone-1.8.2.jar -P /"
COPY ./build/nucleus/entrypoint.sh /
ENTRYPOINT  ["/bin/sh", "/entrypoint.sh"]