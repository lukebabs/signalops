GO_IMAGE ?= golang:1.22-bookworm
PYTHON_IMAGE ?= python:3.12-slim
DOCKER_WORKDIR ?= /workspace
IMAGE ?= signalops-gateway:local
COMPOSE ?= docker compose

.PHONY: docker-test docker-test-python docker-test-broker-integration docker-build docker-build-massive-puller docker-build-massive-scheduler docker-shell docker-validate-schemas compose-up compose-down compose-logs compose-ps compose-validate compose-storage-migrate

docker-test:
	docker run --rm \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		go test ./...

docker-test-python:
	docker run --rm \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		-e PYTHONPATH=$(DOCKER_WORKDIR)/python \
		$(PYTHON_IMAGE) \
		python -m unittest discover -s python/tests

docker-test-broker-integration:
	docker run --rm --network host \
		-e SIGNALOPS_BROKER_INTEGRATION=1 \
		-e SIGNALOPS_BROKER_BROKERS=localhost:19092 \
		-e SIGNALOPS_ENV=local \
		-v $(CURDIR):$(DOCKER_WORKDIR) \
		-w $(DOCKER_WORKDIR) \
		$(GO_IMAGE) \
		go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v

docker-build:
	docker build --target gateway -t $(IMAGE) .

docker-build-massive-puller:
	docker build --target massive-puller -t signalops-massive-puller:local .

docker-build-massive-scheduler:
	docker build --target massive-scheduler -t signalops-massive-scheduler:local .

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

compose-storage-migrate:
	$(COMPOSE) --profile storage run --rm postgres-migrate
