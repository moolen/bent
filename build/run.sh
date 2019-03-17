#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

ROOT=$(dirname "${BASH_SOURCE[0]}")/..
TAG=${TRAVIS_TAG:-"latest"}

docker build -f $ROOT/build/bent-trace-fwd/Dockerfile -t moolen/bent-trace-fwd:latest $ROOT

for DIR in $(find ./build  ! -path ./build -type d); do
    echo $DIR
    REPO=$(basename $DIR)
    docker build -f $ROOT/$DIR/Dockerfile -t moolen/$REPO:$TAG $ROOT
done
