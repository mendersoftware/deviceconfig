FROM golang:1.18.1-alpine3.15 as builder
WORKDIR /go/src/github.com/mendersoftware/deviceconfig
RUN apk add --no-cache \
    xz-dev \
    musl-dev \
    gcc \
    ca-certificates
COPY ./ .
RUN CGO_ENABLED=0 go build

FROM scratch
WORKDIR /etc/deviceconfig
EXPOSE 8080
COPY ./config.yaml .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/mendersoftware/deviceconfig/deviceconfig /usr/bin/

ENTRYPOINT ["/usr/bin/deviceconfig", "--config", "/etc/deviceconfig/config.yaml"]
