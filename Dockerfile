# syntax=docker/dockerfile:1.27.0@sha256:bde3983e9c939224420ddaf6b784cc30e09b035a4dea01f581230c50809f372e

FROM golang:1.27-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS build

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
