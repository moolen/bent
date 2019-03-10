FROM golang:alpine as builder
WORKDIR /go/src/github.com/moolen/bent/
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
RUN apk add --update make glide bash
COPY . .
RUN make build

FROM alpine:3.9
COPY --from=builder /go/src/github.com/moolen/bent/bin/bent /bent
ENTRYPOINT [ "/bent" ]
