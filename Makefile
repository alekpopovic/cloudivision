.PHONY: fmt test vet lint build run-api run-controller run-controller-local run-runner docker-build-api docker-build-controller docker-build-runner docker-build-web manifests generate install uninstall helm-template

IMAGE_REGISTRY ?= ghcr.io/cloudivision
IMAGE_TAG ?= dev
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod
GOTMPDIR ?= $(CURDIR)/.cache/go-tmp
GOFLAGS ?= -p=1
CONTROLLER_GEN ?= controller-gen

export GOCACHE
export GOMODCACHE
export GOTMPDIR
export GOFLAGS

fmt:
	gofmt -w ./api ./cmd ./internal

test:
	@mkdir -p $(GOTMPDIR)
	go test ./...

vet:
	@mkdir -p $(GOTMPDIR)
	go vet ./...

lint:
	@echo "No linters configured yet"

build:
	@mkdir -p $(GOTMPDIR)
	go build -o bin/cloudivision-api ./cmd/api
	go build -o bin/cloudivision-controller ./cmd/controller
	go build -o bin/cloudivision-runner ./cmd/runner
	@if [ -f web/package.json ]; then npm --prefix web run build; fi

run-api:
	@mkdir -p $(GOTMPDIR)
	go run ./cmd/api

run-controller:
	@mkdir -p $(GOTMPDIR)
	go run ./cmd/controller

run-controller-local:
	@mkdir -p $(GOTMPDIR)
	CLOUDIVISION_LEADER_ELECTION=false go run ./cmd/controller

run-runner:
	@mkdir -p $(GOTMPDIR)
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
	@if command -v $(CONTROLLER_GEN) >/dev/null 2>&1; then \
		$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases; \
	else \
		echo "controller-gen is not installed; CRD/RBAC generation skipped."; \
		echo "Run: controller-gen rbac:roleName=manager-role crd webhook paths=\"./...\" output:crd:artifacts:config=config/crd/bases"; \
	fi

generate:
	@if command -v $(CONTROLLER_GEN) >/dev/null 2>&1; then \
		$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."; \
	else \
		echo "controller-gen is not installed; deepcopy generation skipped."; \
		echo "Run: controller-gen object:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""; \
	fi

install: manifests
	kubectl apply -k config/default

uninstall:
	kubectl delete -k config/default --ignore-not-found

helm-template:
	helm template cloudivision charts/cloudivision
