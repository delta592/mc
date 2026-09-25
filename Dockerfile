# syntax=docker/dockerfile:1.27.0@sha256:bde3983e9c939224420ddaf6b784cc30e09b035a4dea01f581230c50809f372e

FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build

LABEL maintainer="MinIO Inc <dev@min.io>"

ENV GOPATH=/go
ENV CGO_ENABLED=0

RUN apk add --no-cache ca-certificates curl && \
    curl -s -q https://raw.githubusercontent.com/delta592/mc/master/LICENSE -o /go/LICENSE && \
    curl -s -q https://raw.githubusercontent.com/delta592/mc/master/CREDITS -o /go/CREDITS

RUN go install -v -ldflags "$(go run buildscripts/gen-ldflags.go)" "github.com/delta592/mc@latest"

FROM scratch

COPY --from=build /go/bin/mc /usr/bin/mc
COPY --from=build /go/CREDITS /licenses/CREDITS
COPY --from=build /go/LICENSE /licenses/LICENSE
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["mc"]
