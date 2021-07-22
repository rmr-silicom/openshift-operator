#!/bin/bash

set -x

base=$(dirname $(realpath "${BASH_SOURCE[0]}"))
bin=$base/bin
PATH=$PATH:$bin
WEB_PORT=8080
ns="intel-fpga-operators"

if $(oc get ns -A | grep -q $ns) ; then
    oc delete ns $ns
fi

oc delete N5010Node worker1 -n intel-fpga-operators

operator-sdk cleanup n5010 --verbose -n $ns

sleep 10

oc create ns $ns

releases=/net/bohr/var/fiberblaze/releases/LightningCreek/ofs-fim/N5010
install_dir=/disks/openshift-provision/install_dir

tar xvf $releases/0_0_1/N5010_ofs-fim_PR_gbs_0_0_1.tar.gz -C $install_dir --wildcards "*_unsigned.bin"

sleep 2

operator-sdk run bundle quay.io/ryan_raasch/intel-fpga-bundle:v2.5.0 --timeout 600s --verbose -n $ns

sleep 15

if ! $(docker ps -a | grep -q static-file-server) ; then
    docker run -d --name static-file-server --rm  -v ${install_dir}:/web -p ${WEB_PORT}:${WEB_PORT} -u $(id -u):$(id -g) halverneus/static-file-server:latest
fi

echo "FIXME: fpga_v1_n5010cluster.yaml doesn't work!!!!"
# oc apply -f $base/N5010/config/samples/fpga_v1_n5010cluster.yaml
oc apply -f $base/N5010/config/samples/fpga_v1_n5010node.yaml
