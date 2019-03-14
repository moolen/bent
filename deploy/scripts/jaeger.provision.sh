#!/bin/bash
set -xe
sudo snap install docker
sleep 5
sudo docker run -d \
  --restart always \
  --name jaeger \
  --net host \
  -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
  jaegertracing/all-in-one:1.10
