# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020 Intel Corporation

VERSION ?= 1.2.1
IMG_VERSION := v$(VERSION)
TLS_VERIFY ?= true

# Operator image registry
IMAGE_REGISTRY ?= registry.connect.redhat.com/intel
# Add suffix directly to IMAGE_REGISTRY to enable empty registry(local images)
ifneq ($(strip $(IMAGE_REGISTRY)),)
override IMAGE_REGISTRY:=$(addsuffix /,$(IMAGE_REGISTRY))
endif

export LABELER_IMAGE ?= $(IMAGE_REGISTRY)n3000-labeler:$(IMG_VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Build labeler binary
labeler: fmt vet
	go build -o bin/labeler ./main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
image:
	cp ../LICENSE TEMP_LICENSE_COPY
	podman build . -t ${LABELER_IMAGE} --build-arg=VERSION=$(IMG_VERSION)

# Push the docker image
push:
	podman push ${LABELER_IMAGE} --tls-verify=$(TLS_VERIFY)

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: fmt vet
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); NAMESPACE=default go test ./... -coverprofile cover.out

build_all: image push
