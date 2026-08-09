#!/bin/bash
#
# Copyright (c) 2015-2023 MinIO, Inc.
#
# This file is part of MinIO Object Storage stack
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

set -euo pipefail

if ! command -v docker >/dev/null; then
	echo "docker is required"
	exit 1
fi

release=$(git describe --abbrev=0 --tags)
root=$(cd "$(dirname "$0")" && pwd)
context=$(mktemp -d)
trap 'rm -rf "$context"' EXIT

LDFLAGS=$(go run "${root}/buildscripts/gen-ldflags.go")
export CGO_ENABLED=0

build_binary() {
	local platform=$1
	local output=$2

	case "${platform}" in
	linux/amd64)
		export GOOS=linux GOARCH=amd64
		unset GOARM GOAMD64
		;;
	*)
		echo "unsupported platform: ${platform}"
		exit 1
		;;
	esac

	mkdir -p "$(dirname "${output}")"
	go build -tags kqueue -trimpath --ldflags "${LDFLAGS}" -o "${output}" .
}

build_binary_v1() {
	local platform=$1
	local output=$2

	case "${platform}" in
	linux/amd64)
		export GOOS=linux GOARCH=amd64 GOAMD64=v1
		unset GOARM
		;;
	*)
		echo "unsupported cpuv1 platform: ${platform}"
		exit 1
		;;
	esac

	mkdir -p "$(dirname "${output}")"
	go build -tags kqueue -trimpath --ldflags "${LDFLAGS}" -o "${output}" .
}

stage_release_context() {
	local dockerfile=$1
	shift
	local platforms=("$@")

	cp "${dockerfile}" "${context}/Dockerfile"
	cp LICENSE CREDITS "${context}/"

	for platform in "${platforms[@]}"; do
		build_binary "${platform}" "${context}/${platform}/mc"
	done
}

sudo sysctl net.ipv6.conf.all.disable_ipv6=1

stage_release_context "${root}/Dockerfile.release" \
	linux/amd64

docker buildx build --push --no-cache \
	-t "delta592/mc:latest" \
	-t "delta592/mc:${release}" \
	-t "quay.io/delta592/mc:${release}" \
	-t "quay.io/delta592/mc:latest" \
	--platform=linux/amd64 \
	"${context}"

docker buildx prune -f

rm -rf "${context}"/*
trap - EXIT
context=$(mktemp -d)
trap 'rm -rf "$context"' EXIT

cp "${root}/Dockerfile.release.old_cpu" "${context}/Dockerfile"
cp LICENSE CREDITS "${context}/"

build_binary_v1 linux/amd64 "${context}/linux/amd64/mc"

docker buildx build --push --no-cache \
	-t "delta592/mc:${release}-cpuv1" \
	-t "quay.io/delta592/mc:${release}-cpuv1" \
	--platform=linux/amd64 \
	"${context}"

docker buildx prune -f

sudo sysctl net.ipv6.conf.all.disable_ipv6=0
