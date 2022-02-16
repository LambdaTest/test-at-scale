NUCLEUS_DOCKER_FILE ?= ./build/nucleus/Dockerfile
NUCLEUS_IMAGE_NAME ?= lambdatest/nucleus:latest

SYNAPSE_DOCKER_FILE ?= ./build/synapse/Dockerfile
SYNAPSE_IMAGE_NAME ?= lambdatest/synapse:latest

usage:						## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/:.*##\s*/##/g' | awk -F'##' '{ printf "%-25s -> %s\n", $$1, $$2 }'

build-nucleus-image:		## builds nucleus docker image
	docker build -t ${NUCLEUS_IMAGE_NAME} --file $(NUCLEUS_DOCKER_FILE) .

build-nucleus-bin:			## builds nucleus binary
	bash build/nucleus/build.sh

build-synapse-image:		## builds synapse docker image
	docker build -t ${SYNAPSE_IMAGE_NAME} --file $(SYNAPSE_DOCKER_FILE) .

build-synapse-bin:			## builds synapse binary
	bash build/synapse/build.sh