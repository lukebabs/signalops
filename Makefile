GO_IMAGE ?= golang:1.22-bookworm
DOCKER_WORKDIR ?= /workspace
IMAGE ?= signalops-gateway:local

.PHONY: docker-test docker-build docker-shell

docker-test:
	docker run --rm \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		go test ./...

docker-build:
	docker build --target gateway -t $(IMAGE) .

docker-shell:
	docker run --rm -it \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		bash

