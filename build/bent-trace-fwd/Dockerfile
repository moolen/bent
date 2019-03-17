FROM golang:alpine as builder
WORKDIR /go/src/github.com/moolen/bent/
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
RUN apk add --update make glide bash git curl gcompat
COPY . .
RUN make build

FROM alpine:3.9
RUN apk add --update libc6-compat
COPY --from=builder /go/src/github.com/moolen/bent/bin/trace-fwd /trace-fwd
ENTRYPOINT [ "/trace-fwd" ]
