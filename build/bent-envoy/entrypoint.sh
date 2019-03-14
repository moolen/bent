#!/bin/bash
set -ex

: ${ENVOY_LOG_LEVEL:=warn}

if [ -z "${ENVOY_NODE_ID}" ]; then
  ENVOY_NODE_ID=$(/usr/local/bin/metadata)
fi

/usr/local/bin/envtemplate \
  -in /etc/envoy/bootstrap.conf.tmpl \
  -out /etc/envoy/bootstrap.conf

ln -sf /dev/stdout /tmp/access.log

/usr/local/bin/envoy \
  --service-node ${ENVOY_NODE_ID} \
  -l ${ENVOY_LOG_LEVEL} \
  -c /etc/envoy/bootstrap.conf \
  $ENVOY_ARGS
