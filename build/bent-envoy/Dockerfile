FROM envoyproxy/envoy-alpine-debug:latest
FROM turbinelabs/envtemplate:0.19.0

FROM golang:alpine as builder
WORKDIR /go/src/github.com/moolen/bent/
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
RUN apk add --update make glide bash
COPY . .
RUN make build

FROM phusion/baseimage:0.10.2
RUN apt-get update
RUN DEBIAN_FRONTEND="noninteractive" apt-get upgrade -y

COPY --from=0 /usr/local/bin/envoy /usr/local/bin/envoy
COPY --from=1 /usr/local/bin/envtemplate /usr/local/bin/envtemplate

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# install jaeger plugin
ENV JAEGER_VERSION=v0.4.2
RUN curl -Lo /usr/local/lib/libjaegertracing_plugin.so \
    https://github.com/jaegertracing/jaeger-client-cpp/releases/download/$JAEGER_VERSION/libjaegertracing_plugin.linux_amd64.so

COPY --from=builder /go/src/github.com/moolen/bent/bin/metadata /usr/local/bin/metadata
ADD build/bent-envoy/bootstrap.conf.tmpl /etc/envoy/bootstrap.conf.tmpl
ADD build/bent-envoy/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

CMD ["/sbin/my_init", "--", "/usr/local/bin/entrypoint.sh"]
