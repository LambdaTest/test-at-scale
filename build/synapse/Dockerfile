FROM golang:latest as builder

# create a working directory
COPY . /synapse
WORKDIR /synapse

# Build binary
RUN go build -o synapse cmd/synapse/*.go

# use a minimal alpine image
FROM docker:latest

ARG VERSION
ENV VERSION=$VERSION

# add ca-certificates in case you need them
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

# Create a group and user
RUN addgroup -S synapse && adduser -S synapse -G synapse

# set working directory
WORKDIR /home/synapse

# copy the binary from builder
COPY --chown=synapse:synapse --from=builder /synapse/synapse .

COPY ./build/synapse/entrypoint.sh /
# run the binary
ENTRYPOINT  ["/bin/sh", "/entrypoint.sh"]