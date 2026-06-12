.PHONY: fmt test vet lint build run-api run-controller run-runner docker-build-api docker-build-controller docker-build-runner docker-build-web manifests generate helm-template

IMAGE_REGISTRY ?= ghcr.io/cloudivision
IMAGE_TAG ?= dev
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod

export GOCACHE
export GOMODCACHE

fmt:
	gofmt -w ./api ./cmd ./internal

test:
	go test ./...

vet:
	go vet ./...

lint:
	@echo "No linters configured yet"

build:
	go build -o bin/cloudivision-api ./cmd/api
	go build -o bin/cloudivision-controller ./cmd/controller
	go build -o bin/cloudivision-runner ./cmd/runner
	@if [ -f web/package.json ]; then npm --prefix web run build; fi

run-api:
	go run ./cmd/api

run-controller:
	go run ./cmd/controller

run-runner:
	go run ./cmd/runner

docker-build-api:
	docker build -f build/api.Dockerfile -t $(IMAGE_REGISTRY)/api:$(IMAGE_TAG) .

docker-build-controller:
	docker build -f build/controller.Dockerfile -t $(IMAGE_REGISTRY)/controller:$(IMAGE_TAG) .

docker-build-runner:
	docker build -f build/runner.Dockerfile -t $(IMAGE_REGISTRY)/runner:$(IMAGE_TAG) .

docker-build-web:
	docker build -f build/web.Dockerfile -t $(IMAGE_REGISTRY)/web:$(IMAGE_TAG) .

manifests:
	@echo "Kubebuilder manifests are not generated yet"

generate:
	@echo "Code generation is not configured yet"

helm-template:
	helm template cloudivision charts/cloudivision
