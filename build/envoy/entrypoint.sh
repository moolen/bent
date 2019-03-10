#!/bin/bash
set -ex

if [ -n "$ENVOY_LOG_LEVEL" ]; then
  ENVOY_ARGS="$ENVOY_ARGS -l $ENVOY_LOG_LEVEL"
fi

if [ -n "$ENVOY_BASE_ID" ]; then
  ENVOY_ARGS="$ENVOY_ARGS --base-id $ENVOY_BASE_ID"
fi

if [ -n "$ENVOY_ADMIN_LOG" ]; then
  mkdir -p $(dirname $ENVOY_ADMIN_LOG)
fi

if [ -n "$ENVOY_LIGHTSTEP_ACCESS_TOKEN" ]; then
  # if the token file is unset, default it to /etc/envoy/lightstep-access-token.
  # this simplifies templating logic to avoid checking for either the token or
  # token_file being set
  if [ -z "${ENVOY_LIGHTSTEP_ACCESS_TOKEN_FILE}" ]; then
    # export so it's available to the envtemplate subprocess
    export ENVOY_LIGHTSTEP_ACCESS_TOKEN_FILE="/etc/envoy/lightstep-access-token"
  fi
  echo $ENVOY_LIGHTSTEP_ACCESS_TOKEN > $ENVOY_LIGHTSTEP_ACCESS_TOKEN_FILE
fi

if [ -z "${SKIP_META}" ]; then
  ENVOY_NODE_ID=$(/usr/local/bin/metadata)
  export ENVOY_NODE_ID
fi

/usr/local/bin/envtemplate \
  -in /etc/envoy/bootstrap.conf.tmpl \
  -out /etc/envoy/bootstrap.conf

ln -sf /dev/stdout /tmp/access.log

echo /usr/local/bin/envoy -c /etc/envoy/bootstrap.conf $ENVOY_ARGS
/usr/local/bin/envoy -c /etc/envoy/bootstrap.conf $ENVOY_ARGS
