
# Image URL to use all building/pushing image targets
IMG ?= enterprise-contract-controller:latest
DOCKER_CONFIG ?= $(HOME)

ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

CONTROLLER_GEN = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/controller-tools/cmd/controller-gen
KUSTOMIZE = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/kustomize/kustomize/v4
ENVTEST = go run -modfile $(ROOT)tools/go.mod sigs.k8s.io/controller-runtime/tools/setup-envtest
CRD_DEF = ./api/v1alpha1

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

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out
	cd api && go test ./... -coverprofile ../api_cover.out
	cd schema && go test ./... -coverprofile ../schema_cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
ifeq ("$(shell docker info --format '{{$$found:=false}}{{range .ClientInfo.Plugins}}{{if eq .Name "buildx"}}{{$$found = true}}{{end}}{{end}}{{if $$found}}true{{else}}false{{end}}')","true")
	docker buildx create --use
	docker buildx build --load -t ${IMG} --cache-from=type=local,src=/tmp/.buildx-cache --cache-to=type=local,dest=/tmp/.buildx-cache,mode=max .
	docker buildx stop
	docker buildx rm
else
	docker build -t ${IMG} .
endif

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

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
