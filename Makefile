# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020-2021 Intel Corporation
include makefile.top

.PHONY: $(TARGETS) tar

export GOPATH=$(shell echo $$HOME)/go

TARGETS := N5010 prometheus_fpgainfo_exporter
ifeq ($(BUILD_SRIO_FEC),yes)
	TARGETS += sriov-fec
endif

IMAGE_TARGETS := $(TARGETS:=-image)
PUSH_TARGETS := $(TARGETS:=-push)

all: $(TARGETS) $(IMAGE_TARGETS) $(PUSH_TARGETS) build_index

$(TARGETS):
	make -C $@ build_all

$(IMAGE_TARGETS):
	make -C $(subst -image,,$@) image

tar-others:
	- rm -rf archives
	mkdir archives
	docker save $(IMAGE_REGISTRY)/dfl-kmod-drivercontainer:eea9cbc-4.18.0-193.el8.x86_64 | gzip > archives/dfl-kmod-drivercontainer.tar.gz
	docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
	docker tag gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0 $(IMAGE_REGISTRY)/kube-rbac-proxy:v0.5.0
	docker save $(IMAGE_REGISTRY)/kube-rbac-proxy:v0.5.0 | gzip > archives/kube-rbac-proxy.tar.gz

tar: tar-others
	for img in $(IMAGES_TAR) ; do \
	echo $(IMAGE_REGISTRY)/$$img:$(VERSION); \
	docker save $(IMAGE_REGISTRY)/$$img:$(VERSION) | gzip > archives/$$img.tar.gz ; \
	done
	tar -czvf intel-fpga-bundle.tar.gz archives

$(PUSH_TARGETS):
	make -C $(subst -push,,$@) push

build_index: $(TARGETS)
ifeq ($(BUILD_SRIO_FEC),yes)
	$(PWD)/bin/opm index add --bundles $(IMAGE_REGISTRY)/sriov-fec-bundle:$(VERSION),$(IMAGE_REGISTRY)/intel-fpga-bundle:$(VERSION) --tag localhost/intel-fpga-operators-index:$(VERSION) $(if ifeq $(TLS_VERIFY) false, --skip-tls) -c podman --mode=semver
	$(PODMAN) push localhost/intel-fpga-operators-index:$(VERSION) $(IMAGE_REGISTRY)/intel-fpga-operators-index:$(VERSION)
endif

clean-tools:
	rm -rf downloads bin

OPERATOR_SDK_INSTALLED := $(shell command -v bin/operator-sdk version 2> /dev/null)
check-operator-sdk-version:
ifndef OPERATOR_SDK_INSTALLED
	$(info operator-sdk is not installed - downloading it)
	scripts/install-tools.sh $(REQUIRED_OPERATOR_SDK_VERSION)
else
ifneq ($(shell bin/operator-sdk version | awk -F',' '{print $$1}' | awk -F'[""]' '{print $$2}'), $(REQUIRED_OPERATOR_SDK_VERSION))
	$(info updating operator-sdk to $(REQUIRED_OPERATOR_SDK_VERSION))
	scripts/install-tools.sh $(REQUIRED_OPERATOR_SDK_VERSION)
endif
endif