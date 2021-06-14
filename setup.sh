#!/bin/bash

set -x

base=$(dirname $(realpath "${BASH_SOURCE[0]}"))

WEB_PORT=8080
ns="intel-fpga-operators"

if $(oc get ns -A | grep -q $ns) ; then
    oc delete ns $ns
fi

/disks/openshift-operator-n3000/bin/operator-sdk cleanup n3000 --verbose -n $ns

sleep 10

$base/bin/oc create ns $ns

releases=/net/bohr/var/fiberblaze/releases/LightningCreek/ofs-fim/N5010
install_dir=/disks/openshift-provision/install_dir

tar xvf $releases/0_0_1/N5010_ofs-fim_PR_gbs_0_0_1.tar.gz -C $install_dir --wildcards "*_unsigned.bin"

sleep 2

$base/bin/operator-sdk run bundle quay.io/ryan_raasch/intel-fpga-bundle:v2.0.0 --timeout 300s --verbose -n $ns

sleep 15

if ! $(docker ps -a | grep -q static-file-server) ; then
    docker run -d --name static-file-server --rm  -v ${install_dir}:/web -p ${WEB_PORT}:${WEB_PORT} -u $(id -u):$(id -g) halverneus/static-file-server:latest
fi

$base/bin/oc apply -f $base/N3000/config/samples/fpga_v1_n3000cluster.yaml
