#!/usr/bin/env bash
docker pull openshift/origin-control-plane:v3.11

iidfile=$(mktemp -t ocf-iidfile.XXXXXXXXXX)
iid=$(cat $iidfile)
containerID=$(docker create $iid openshift/origin-control-plane:v3.11)
docker cp $containerID:/usr/bin/oc bin/oc
docker container rm $containerID
