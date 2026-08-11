# syntax=docker/dockerfile:1.26.0@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32

FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS build

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
