FROM golang:1.16.3-alpine3.12 as builder
RUN apk add --no-cache \
    xz-dev \
    musl-dev \
    gcc
RUN mkdir -p /go/src/github.com/mendersoftware/deviceconfig
COPY . /go/src/github.com/mendersoftware/deviceconfig
RUN cd /go/src/github.com/mendersoftware/deviceconfig && env CGO_ENABLED=1 go build

FROM alpine:3.13.5
RUN apk add --no-cache ca-certificates xz
RUN mkdir -p /etc/deviceconfig
COPY ./config.yaml /etc/deviceconfig
COPY --from=builder /go/src/github.com/mendersoftware/deviceconfig/deviceconfig /usr/bin
ENTRYPOINT ["/usr/bin/deviceconfig", "--config", "/etc/deviceconfig/config.yaml"]

EXPOSE 8080
