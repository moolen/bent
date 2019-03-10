#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

ROOT=$(dirname "${BASH_SOURCE[0]}")/..

docker build -f $ROOT/build/envoy/Dockerfile     -t moolen/bent-envoy:latest $ROOT
docker build -f $ROOT/build/trace-fwd/Dockerfile -t moolen/trace-fwd:latest $ROOT
docker build -f $ROOT/build/bent/Dockerfile      -t moolen/bent:latest $ROOT
