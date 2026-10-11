# syntax=docker/dockerfile:1.28.0@sha256:bb22d9815c728170f72750f4e5b0d672e06176142e1d602c7e66c050100b7e5b

FROM golang:1.27-alpine@sha256:4cb7ac979db5fcc41cae44b2227ba5ab8a51e8807f40d9ba4dee20a0ad960b5b AS build

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
