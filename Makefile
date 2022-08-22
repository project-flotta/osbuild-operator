# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# osbuilder.project-flotta.io/osbuild-operator-bundle:$VERSION and osbuilder.project-flotta.io/osbuild-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= osbuilder.project-flotta.io/osbuild-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# Worker setup job container
SETUP_IMG ?= quay.io/project-flotta/osbuild-operator-worker-setup:v0.1
SETUP_DOCKERFILE = Dockerfile.WorkerSetupJob

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

CERT_MANAGER_VERSION = v1.9.0

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

# Docker command to use, can be podman
ifneq (, $(shell which podman))
CONTAINER_RUNTIME := podman
else
ifneq (, $(shell which docker))
CONTAINER_RUNTIME := docker
else
$(error "Neither docker nor podman are available in PATH")
endif
endif

# Kubectl command to use, can be oc
ifneq (, $(shell which kubectl))
CLUSTER_RUNTIME := kubectl
else
ifneq (, $(shell which oc))
CLUSTER_RUNTIME := oc
else
$(error "Neither kubectl nor oc are available in PATH")
endif
endif

.PHONY: all
all: build

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

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate-tools
generate-tools:
ifeq (, $(shell which mockery))
	(cd /tmp && go install github.com/vektra/mockery/...@v1.1.2)
endif
ifeq (, $(shell which mockgen))
	(cd /tmp/ && go install github.com/golang/mock/mockgen@v1.6.0)
endif
	@exit

.PHONY: generate
generate: generate-tools controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	go generate ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: gosec
gosec: ## Run gosec locally
	$(CONTAINER_RUNTIME) run --rm -v $(PWD):/opt/data/:z docker.io/securego/gosec -exclude-generated /opt/data/...

GO_IMAGE=golang:1.17.8-alpine3.14
GOIMPORTS_IMAGE=golang.org/x/tools/cmd/goimports@latest
FILES_LIST=$(shell ls -d */ | grep -v -E "vendor|tools|test|client|restapi|models|generated")
MODULE_NAME=$(shell head -n 1 go.mod | cut -d '/' -f 3)
.PHONY: imports
imports: ## fix and format go imports
	@# Removes blank lines within import block so that goimports does its magic in a deterministic way
	find $(FILES_LIST) -type f -name "*.go" | xargs -L 1 sed -i '/import (/,/)/{/import (/n;/)/!{/^$$/d}}'
	$(CONTAINER_RUNTIME) run --rm -v $(CURDIR):$(CURDIR) -w="$(CURDIR)" $(GO_IMAGE) \
		sh -c 'go install $(GOIMPORTS_IMAGE) && goimports -w -local github.com/project-flotta $(FILES_LIST) && goimports -w -local github.com/project-flotta/$(MODULE_NAME) $(FILES_LIST)'

LINT_IMAGE=golangci/golangci-lint:v1.45.0
.PHONY: lint
lint: ## Check if the go code is properly written, rules are in .golangci.yml
	$(CONTAINER_RUNTIME) run --rm -v $(CURDIR):$(CURDIR) -w="$(CURDIR)" $(LINT_IMAGE) sh -c 'golangci-lint run'

.PHONY: test
test: ## Run tests
test: manifests generate fmt vet envtest
test: test-fast

TEST_PACKAGES := ./...
test-fast: ginkgo
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) --cover -output-dir=. -coverprofile=cover.out -v -progress $(GINKGO_OPTIONS) $(TEST_PACKAGES)

test-create-coverage:
	sed -i '/mock_/d' cover.out
	sed -i '/zz_generated/d' cover.out
	go tool cover -func cover.out
	go tool cover --html=cover.out -o coverage.html

test-coverage:
	go tool cover --html=cover.out

.PHONY: vendor
vendor:
	go mod tidy -go=1.17
	go mod vendor


build-iso: ## build fake iso
	mkisofs -o testdata/fake_iso.iso testdata/iso_source

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -mod=vendor -o bin/manager main.go
	go build -mod=vendor -o bin/httpapi ./cmd/httpapi/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	$(CONTAINER_RUNTIME) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_RUNTIME) push ${IMG}

.PHONY: setup-image-build
setup-image-build: # Build container image for the Worker setup job
	$(CONTAINER_RUNTIME) build -t ${SETUP_IMG} -f ${SETUP_DOCKERFILE} .

.PHONY: setup-image-push
setup-image-push: ## Push the Worker setup job image.
	$(CONTAINER_RUNTIME) push ${SETUP_IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(CLUSTER_RUNTIME) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(CLUSTER_RUNTIME) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: install-cert-manager
install-cert-manager: cmctl
	$(CLUSTER_RUNTIME) apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml
	${CMCTL} check api --wait=5m

.PHONY: deploy
deploy: manifests kustomize install-cert-manager ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd config/httpapi && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(CLUSTER_RUNTIME) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(CLUSTER_RUNTIME) delete --ignore-not-found=$(ignore-not-found) -f -
	${CLUSTER_RUNTIME} delete --ignore-not-found=$(ignore-not-found) -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@latest)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

CMCTL = $(shell pwd)/bin/cmctl
OS = $(shell go env GOOS)
ARCH = $(shell go env GOARCH)
.PHONY: cmctl
cmctl: ## Download cmctl locally if necessary.
ifeq ("$(wildcard $(CMCTL))","")
	curl -sSL -o /tmp/cmctl.tar.gz https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cmctl-${OS}-${ARCH}.tar.gz
	tar xzf /tmp/cmctl.tar.gz -C ./bin ./cmctl
endif

# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(CONTAINER_RUNTIME) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.19.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif


minio-certs-generate:
	# Secrets are already created in a file, but expiration it's a year. Instead of
	# creating a new one every pr, it's much better to have the secret file, and if
	# no longer works, create a new secret file using this cert.
	curl -L https://github.com/minio/certgen/releases/latest/download/certgen-linux-amd64 -o testdata/certgen
	chmod 777 testdata/certgen
	./testdata/certgen -host "*" -duration "876600h0m0s"
	kubectl create secret generic tls-ssl-minio --from-file=private.key --from-file=public.crt --dry-run=client -o yaml > testdata/minio/secret.yaml

minio-install: ## Install Minio on the current kubectl config
	kubectl create ns minio || exit 0
	kubectl -n minio apply -f testdata/minio/secret.yaml
	kubectl -n minio apply -f testdata/minio/minio.yaml
	kubectl wait --for=condition=Ready pods --all -n minio --timeout=60s

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

get-iso:
ifeq (,$(wildcard ./testdata/Fedora.iso))
	curl -L https://download.fedoraproject.org/pub/fedora/linux/releases/36/Server/x86_64/iso/Fedora-Server-netinst-x86_64-36-1.5.iso -o testdata/Fedora.iso
endif

GINKGO = $(shell pwd)/bin/ginkgo
.PHONY: ginkgo
ginkgo: ## Download ginkgo locally if necessary.
ifeq (, $(wildcard $(GINKGO)))
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.3)
endif

.PHONY: composer-api
update-composer-api:
	curl -o internal/composer/openapi.v2.yml https://raw.githubusercontent.com/osbuild/osbuild-composer/main/internal/cloudapi/v2/openapi.v2.yml
