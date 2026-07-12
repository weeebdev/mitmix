REGISTRY ?= ghcr.io
IMAGE   ?= $(REGISTRY)/$(shell whoami)/mitmix
TAG     ?= latest

.PHONY: all hub single test

all: hub

hub:
	docker build -t $(IMAGE)-hub:$(TAG) -f Dockerfile.hub .

agent:
	docker build -t $(IMAGE)-agent:$(TAG) -f agent/Dockerfile.agent .

single:
	docker build -t $(IMAGE):$(TAG) -f Dockerfile.single .

test:
	cd internal/hub/site && npm run build
	nix develop --command go test ./...
	cd agent && .venv/bin/python -m pytest tests/ -q

push-single: single
	docker push $(IMAGE):$(TAG)
