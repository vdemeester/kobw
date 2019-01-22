#!/usr/bin/env bash
set -e
if [[ $TRAVIS_PULL_REQUEST == "false" ]] && [[ $TRAVIS_BRANCH == "master" ]]; then
    docker push docker.io/vdemeester/kobw-base
    docker push docker.io/vdemeester/kobw-builder
    docker push quay.io/vdemeest/kobw-base
    docker push quay.io/vdemeest/kobw-builder
fi

