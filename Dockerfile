# syntax=docker/dockerfile:1.26.0@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32

FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS build

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
