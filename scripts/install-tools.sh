#!/bin/bash

set -xe

BASE=$(dirname $(realpath "${BASH_SOURCE[0]}"))
BIN=$(realpath $BASE/../bin)
GOPATH=$(realpath $BASE/../bin)
DOWNLOADS=$(realpath $BASE/../downloads)
REQUIRED_OPERATOR_SDK_VERSION="${1:-v1.4.2}"
SDK_URL="https://github.com/operator-framework/operator-sdk/releases/download"
OPM_URL="https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/stable-4.6"

OC_URL="https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/stable-4.6"
OC_FILE="openshift-client-linux.tar.gz"

if [ ! -e $BIN ] ; then
    mkdir -p $BIN
fi

if [ ! -e $DOWNLOADS ] ; then
    mkdir -p $DOWNLOADS
fi

if [ ! -e $BIN/operator-sdk ] ; then
    curl -sL $SDK_URL/$REQUIRED_OPERATOR_SDK_VERSION/operator-sdk_linux_amd64 -o $BIN/operator-sdk
    chmod +x $BIN/operator-sdk
fi

if [ ! -e $BIN/opm ] ; then
    for rev in $(seq 33 100); do
        curl -sL $OPM_URL/opm-linux-4.6.$rev.tar.gz -o $DOWNLOADS/opm.tar.gz
        if $(file $DOWNLOADS/opm.tar.gz | grep -q "gzip compressed data") ; then
            tar xvf $DOWNLOADS/opm.tar.gz -C $BIN
            break
        fi
    done
fi

if [ ! -e $BIN/oc ] ; then
    curl -sL $OC_URL/$OC_FILE -o $DOWNLOADS/$OC_FILE
    tar xvf $DOWNLOADS/$OC_FILE -C $BIN
fi

[ ! -e $GOPATH ] && mkdir -p $GOPATH

GOBIN=$BIN GOPATH=$GOPATH GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1
GOBIN=$BIN GOPATH=$GOPATH GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize

chmod -R +w $GOPATH/*
