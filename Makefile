GO_IMAGE ?= golang:1.22-bookworm
PYTHON_IMAGE ?= python:3.12-slim
DOCKER_WORKDIR ?= /workspace
IMAGE ?= signalops-gateway:local
COMPOSE ?= docker compose

.PHONY: docker-test docker-build docker-shell docker-validate-schemas compose-up compose-down compose-logs compose-ps compose-validate

docker-test:
	docker run --rm \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		go test ./...

docker-build:
	docker build --target gateway -t $(IMAGE) .

docker-validate-schemas:
	docker run --rm \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(PYTHON_IMAGE) \
		python scripts/validate_json_schemas.py

docker-shell:
	docker run --rm -it \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		bash


compose-up:
	$(COMPOSE) up -d --build

compose-down:
	$(COMPOSE) down

compose-logs:
	$(COMPOSE) logs -f

compose-ps:
	$(COMPOSE) ps

compose-validate:
	$(COMPOSE) config --quiet
