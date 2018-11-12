#!/usr/bin/env bash
if [[ $TRAVIS_PULL_REQUEST == "false" ]] && [[ $TRAVIS_BRANCH == "master" ]]; then
    docker push docker.io/vdemeester/kobw-base
    docker push docker.io/vdemeester/kobw-builder
fi

