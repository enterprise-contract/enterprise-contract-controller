# Image URL to use all building/pushing image targets
IMG ?= enterprise-contract-controller:latest
DOCKER_CONFIG ?= $(HOME)

ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

CONTROLLER_GEN = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/controller-tools/cmd/controller-gen
KUSTOMIZE = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/kustomize/kustomize/v4
ENVTEST = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/controller-runtime/tools/setup-envtest
CRD_DEF = ./api/v1alpha1

# Test related variables
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
ENVTEST_K8S_VERSION=1.29.0
TEKTON_VERSION=v0.57.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build manifests docs

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: docs
docs: $(wildcard $(CRD_DEF)/*.go) ## Generate documentation
	@go run -modfile tools/go.mod github.com/elastic/crd-ref-docs --max-depth 50 --config=docs/config.yaml --source-path=$(CRD_DEF) --templates-dir=docs/templates --output-path=docs/modules/ROOT/pages/reference.adoc
	@go run ./docs

GEN_DEPS=\
 controllers/enterprisecontractpolicy_controller.go \
 api/v1alpha1/enterprisecontractpolicy_types.go \
 api/v1alpha1/groupversion_info.go \
 tools/go.sum

config/crd/bases/%.yaml: $(GEN_DEPS)
	$(CONTROLLER_GEN) rbac:roleName=enterprise-contract-role crd webhook paths=./... output:crd:artifacts:config=config/crd/bases
	yq -i 'del(.metadata.annotations["controller-gen.kubebuilder.io/version"])' $@

api/config/%.yaml: config/crd/bases/%.yaml
	@mkdir -p api/config
	@cp $< $@

manifests: api/config/appstudio.redhat.com_enterprisecontractpolicies.yaml ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.

.PHONY: generate
generate: $(GEN_DEPS) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths=./...
	cd api && go generate ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test-setup
test-setup: ## Download envtest-setup locally if necessary.
	@echo "Setting up test environment..."
	@if [ ! -f $(GOBIN)/setup-envtest ]; then \
		echo "Installing setup-envtest..."; \
		go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest; \
	fi
	@echo "Downloading envtest binaries..."
	@$(GOBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(ENVTEST_ASSETS_DIR)
	@echo "Test environment setup complete"

.PHONY: download-tekton-crds
download-tekton-crds: ## Download Tekton CRDs for testing
	@echo "Downloading Tekton CRDs..."
	@mkdir -p config/crd/tekton
	@curl -sL https://github.com/tektoncd/pipeline/releases/download/$(TEKTON_VERSION)/release.yaml > config/crd/tekton/release.yaml
	@echo "Extracting CRDs..."
	@awk '/kind: CustomResourceDefinition/,/^---/' config/crd/tekton/release.yaml > config/crd/tekton/crds.yaml
	@rm config/crd/tekton/release.yaml

.PHONY: test
test: test-setup download-tekton-crds ## Run tests
	KUBEBUILDER_ASSETS=$(ENVTEST_ASSETS_DIR)/k8s/$(ENVTEST_K8S_VERSION)-darwin-arm64 go test ./controllers/... -v

.PHONY: test-clean
test-clean: ## Clean up test artifacts
	@echo "Cleaning up test artifacts..."
	@rm -f config/crd/tekton/crds.yaml
	@rm -f config/crd/tekton/release.yaml
	@echo "Cleanup complete"

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build container image with the manager.
	podman build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push container image with the manager.
	podman push ${IMG}

.PHONY: export-schema
export-schema: generate ## Export the CRD schema to the schema directory as a json-store.org schema.
	@mkdir -p dist
	cp api/v1alpha1/policy_spec.json dist/

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
