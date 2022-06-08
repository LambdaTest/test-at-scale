FROM golang:latest as builder

# create a working directory
COPY . /nucleus
WORKDIR /nucleus


# Build binary
RUN GOARCH=amd64 GOOS=linux go build -ldflags="-w -s" -o nucleus cmd/nucleus/*.go
# Uncomment only when build is highly stable. Compress binary.
# RUN strip --strip-unneeded ts
# RUN upx ts

FROM golang:latest

RUN apt update && apt install -y git zstd chromium curl

COPY --from=golang:latest /usr/local/go/ /usr/local/go/

RUN useradd -u 8877 -ms /bin/bash nucleus

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
ENTRYPOINT  ["/bin/sh", "/entrypoint.sh"]
