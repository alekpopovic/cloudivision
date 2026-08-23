.PHONY: help fmt test test-unit test-controller test-api test-web test-e2e test-all conformance upgrade-test scale-test security-check release-local vet lint build build-cli install-cli-local cli-completions run-api run-controller run-controller-local run-runner docker-build-api docker-build-controller docker-build-runner docker-build-web manifests generate sync-chart-crds install uninstall helm-template

IMAGE_REGISTRY ?= ghcr.io/alekpopovic/cloudivision
IMAGE_TAG ?= dev
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod
GOTMPDIR ?= $(CURDIR)/.cache/go-tmp
CODEGEN_GOMODCACHE ?= /tmp/cloudivision-go-mod
GOFLAGS ?= -p=1
VERSION ?= $(shell tr -d '[:space:]' < VERSION)
GIT_COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
CLI_LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.buildDate=$(BUILD_DATE)
CONTROLLER_GEN ?= controller-gen
CONTROLLER_GEN_API_PATHS ?= ./api/...
CONTROLLER_GEN_MANIFEST_PATHS ?= ./api/...;./internal/controller/...

export GOCACHE
export GOMODCACHE
export GOTMPDIR
export GOFLAGS

help:
	@echo "cloudivision development targets"
	@echo "  make test          Run all Go tests"
	@echo "  make test-all      Run Go, controller, API and web checks"
	@echo "  make vet           Run go vet"
	@echo "  make build         Build Go binaries and the Angular UI"
	@echo "  make build-cli     Build the versioned cloudivision CLI"
	@echo "  make install-cli-local Install CLI into GOPATH/bin"
	@echo "  make cli-completions Generate shell completions in dist/completions"
	@echo "  make manifests     Regenerate CRDs and RBAC with controller-gen"
	@echo "  make sync-chart-crds Refresh the Helm CRD bundle from generated bases"
	@echo "  make helm-template Render and security-check the Helm chart"
	@echo "  make install       Install manifests into the current kubectl context"
	@echo "  make uninstall     Remove manifests from the current kubectl context"
	@echo "  make test-e2e      Run the kind smoke-test entrypoint"
	@echo "  make conformance   Run clean-cluster platform conformance scenarios"
	@echo "  make upgrade-test  Validate upgrade assets (set UPGRADE_TEST_LIVE=true for a disposable cluster)"
	@echo "  make scale-test    Generate a limited scale fixture (set SCALE_TEST_LIVE=true to apply)"
	@echo "  make security-check Check the rendered chart runner security baseline"
	@echo "  make release-local  Build local versioned release artifacts"

fmt:
	gofmt -w ./api ./cmd ./internal

test:
	@mkdir -p $(GOTMPDIR)
	go test ./...

test-unit:
	@mkdir -p $(GOTMPDIR)
	go test ./api/... ./internal/domain/... ./internal/webhook/... ./internal/build/... ./internal/gitops/... ./internal/executor/... ./internal/audit/... ./internal/auth/... ./internal/redact/... ./internal/runner/...

test-controller:
	@mkdir -p $(GOTMPDIR)
	@if [ -z "$${KUBEBUILDER_ASSETS:-}" ]; then \
		echo "KUBEBUILDER_ASSETS is not set; running controller fake-client tests only. Set KUBEBUILDER_ASSETS to enable future envtest-backed tests."; \
	fi
	go test ./internal/controller/...

test-api:
	@mkdir -p $(GOTMPDIR)
	go test ./internal/api/...

test-web:
	npm --prefix web test -- --watch=false

test-e2e:
	./test/e2e/kind_smoke.sh

conformance:
	./test/conformance/run.sh

upgrade-test:
	./test/upgrade/run.sh

scale-test:
	./test/scale/run.sh

security-check:
	@rendered="$$(mktemp)"; trap 'rm -f "$$rendered"' EXIT; \
		helm template cloudivision charts/cloudivision --include-crds > "$$rendered"; \
		./test/security/no-privileged.sh "$$rendered"; \
		./test/security/no-docker-sock.sh "$$rendered"; \
		./test/security/no-hostpath.sh "$$rendered"; \
		./test/security/no-invalid-pod-security-context.sh "$$rendered"; \
		./test/security/rbac-minimal.sh "$$rendered"

release-local:
	./scripts/release/build-local.sh

test-all: fmt test-unit test-controller test-api test-web vet build helm-template

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
	$(MAKE) build-cli
	@if [ -f web/package.json ]; then npm --prefix web run build; fi

build-cli:
	@mkdir -p $(GOTMPDIR) bin
	go build -trimpath -ldflags "$(CLI_LDFLAGS)" -o bin/cloudivision ./cmd/cloudivision

install-cli-local:
	@mkdir -p $(GOTMPDIR)
	go install -trimpath -ldflags "$(CLI_LDFLAGS)" ./cmd/cloudivision

cli-completions: build-cli
	@mkdir -p dist/completions
	@for shell in bash zsh fish powershell; do bin/cloudivision completion $$shell > dist/completions/cloudivision.$$shell; done

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
		GOMODCACHE=$(CODEGEN_GOMODCACHE) $(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="$(CONTROLLER_GEN_MANIFEST_PATHS)" output:crd:artifacts:config=config/crd/bases; \
		./hack/sync-chart-crds.sh; \
	else \
		echo "controller-gen is not installed; CRD/RBAC generation skipped."; \
		echo "Run: controller-gen rbac:roleName=manager-role crd webhook paths=\"$(CONTROLLER_GEN_MANIFEST_PATHS)\" output:crd:artifacts:config=config/crd/bases"; \
	fi

sync-chart-crds:
	./hack/sync-chart-crds.sh

generate:
	@if command -v $(CONTROLLER_GEN) >/dev/null 2>&1; then \
		GOMODCACHE=$(CODEGEN_GOMODCACHE) $(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="$(CONTROLLER_GEN_API_PATHS)"; \
	else \
		echo "controller-gen is not installed; deepcopy generation skipped."; \
		echo "Run: controller-gen object:headerFile=\"hack/boilerplate.go.txt\" paths=\"$(CONTROLLER_GEN_API_PATHS)\""; \
	fi

install: manifests
	kubectl apply -k config/default

uninstall:
	kubectl delete -k config/default --ignore-not-found

helm-template:
	helm template cloudivision charts/cloudivision --include-crds
	@if helm template cloudivision charts/cloudivision --include-crds | grep -q "privileged: true"; then \
		echo "Rendered chart contains privileged: true"; \
		exit 1; \
	fi
