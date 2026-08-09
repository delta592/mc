#!/usr/bin/env bash
#
# Run the mc integration test suite, starting a local MinIO server when needed.
# If MinIO is already reachable at the configured endpoint, it is left running.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

BUILD_TAGS="${BUILD_TAGS:-kqueue}"
INTEGRATION_TESTPKG="${INTEGRATION_TESTPKG:-./cmd}"
TEST_TIMEOUT="${TEST_TIMEOUT:-20m}"
RACE_TEST_FLAGS="-race -tags ${BUILD_TAGS} -count=1 -timeout ${TEST_TIMEOUT}"

normalize_https() {
	local val
	val="$(echo "$1" | tr '[:upper:]' '[:lower:]')"
	case "$val" in
	1 | true | yes) echo "true" ;;
	*) echo "false" ;;
	esac
}

# Accept CI-style env vars as aliases for MC_TEST_* settings.
SERVER_ENDPOINT="${MC_TEST_SERVER_ENDPOINT:-${SERVER_ENDPOINT:-127.0.0.1:9000}}"
MC_TEST_SERVER_ENDPOINT="$SERVER_ENDPOINT"
ACCESS_KEY="${MC_TEST_ACCESS_KEY:-${ACCESS_KEY:-minioadmin}}"
MC_TEST_ACCESS_KEY="$ACCESS_KEY"
SECRET_KEY="${MC_TEST_SECRET_KEY:-${SECRET_KEY:-minioadmin}}"
MC_TEST_SECRET_KEY="$SECRET_KEY"
ENABLE_HTTPS="$(normalize_https "${MC_TEST_ENABLE_HTTPS:-${ENABLE_HTTPS:-false}}")"
MC_TEST_ENABLE_HTTPS="$ENABLE_HTTPS"
SKIP_INSECURE="${MC_TEST_SKIP_INSECURE:-true}"
RUN_FUNCTIONAL="${MC_TEST_RUN_FUNCTIONAL:-false}"
INSTALL_CA="${MC_TEST_INSTALL_CA:-false}"

if [ "$ENABLE_HTTPS" = "true" ]; then
	PROTOCOL="https"
else
	PROTOCOL="http"
fi

HEALTH_URL="${PROTOCOL}://${SERVER_ENDPOINT}/minio/health/live"
MINIO_BIN="${ROOT_DIR}/.cache/minio/minio"
MINIO_DATA_DIR=""
MINIO_PID=""
MINIO_STARTED_BY_US=false

curl_health() {
	local curl_args=(-sf --max-time 2)
	if [ "$SKIP_INSECURE" = "true" ] && [ "$PROTOCOL" = "https" ]; then
		curl_args+=(-k)
	fi
	curl "${curl_args[@]}" "$HEALTH_URL" >/dev/null 2>&1
}

wait_for_minio() {
	local attempt
	for attempt in $(seq 1 30); do
		if curl_health; then
			return 0
		fi
		sleep 1
	done
	echo "Timed out waiting for MinIO at ${HEALTH_URL}" >&2
	return 1
}

minio_platform() {
	local os arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "$arch" in
	x86_64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	esac
	echo "${os}-${arch}"
}

ensure_minio_binary() {
	if [ -x "$MINIO_BIN" ]; then
		return 0
	fi

	mkdir -p "$(dirname "$MINIO_BIN")"
	local platform url
	platform="$(minio_platform)"
	url="https://dl.min.io/server/minio/release/${platform}/minio"

	echo "Downloading MinIO binary for ${platform}..."
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$MINIO_BIN" "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$MINIO_BIN" "$url"
	else
		echo "curl or wget is required to download MinIO" >&2
		return 1
	fi
	chmod +x "$MINIO_BIN"
}

install_system_ca() {
	if [ "$INSTALL_CA" != "true" ] || [ "$ENABLE_HTTPS" != "true" ]; then
		return 0
	fi
	if [ "$(uname -s)" != "Linux" ]; then
		return 0
	fi
	if ! command -v sudo >/dev/null 2>&1; then
		echo "sudo is required to install the MinIO CA certificate" >&2
		return 1
	fi

	echo "Installing MinIO TLS certificate into system trust store..."
	sudo cp "${ROOT_DIR}/testdata/localhost.crt" /usr/local/share/ca-certificates/localhost.crt
	sudo update-ca-certificates
}

configure_minio_tls() {
	if [ "$ENABLE_HTTPS" != "true" ]; then
		return 0
	fi

	mkdir -p "${HOME}/.minio/certs"
	cp "${ROOT_DIR}/testdata/localhost.crt" "${HOME}/.minio/certs/public.crt"
	cp "${ROOT_DIR}/testdata/localhost.key" "${HOME}/.minio/certs/private.key"
}

run_functional_tests() {
	if [ "$RUN_FUNCTIONAL" != "true" ]; then
		return 0
	fi

	echo "Running functional tests"
	export SERVER_ENDPOINT="$MC_TEST_SERVER_ENDPOINT"
	export ACCESS_KEY="$MC_TEST_ACCESS_KEY"
	export SECRET_KEY="$MC_TEST_SECRET_KEY"
	if [ "$ENABLE_HTTPS" = "true" ]; then
		export ENABLE_HTTPS=1
	else
		export ENABLE_HTTPS=0
	fi
	env bash "${ROOT_DIR}/functional-tests.sh"
}

start_minio() {
	MINIO_DATA_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mc-integration-minio.XXXXXX")"
	mkdir -p "${MINIO_DATA_DIR}/data"

	configure_minio_tls

	echo "Starting MinIO at ${PROTOCOL}://${SERVER_ENDPOINT}..."
	MINIO_ROOT_USER="$ACCESS_KEY" MINIO_ROOT_PASSWORD="$SECRET_KEY" \
		"$MINIO_BIN" server "${MINIO_DATA_DIR}/data" --address "${SERVER_ENDPOINT}" \
		>"${MINIO_DATA_DIR}/minio.log" 2>&1 &
	MINIO_PID=$!
	MINIO_STARTED_BY_US=true

	wait_for_minio
	echo "MinIO is ready"
}

ensure_minio() {
	if curl_health; then
		echo "Using existing MinIO server at ${HEALTH_URL}"
		return 0
	fi

	ensure_minio_binary
	start_minio
}

stop_minio() {
	if [ "$MINIO_STARTED_BY_US" != true ] || [ -z "$MINIO_PID" ]; then
		return 0
	fi

	echo "Stopping MinIO started for integration tests (pid ${MINIO_PID})..."
	kill "$MINIO_PID" 2>/dev/null || true
	wait "$MINIO_PID" 2>/dev/null || true

	if [ -n "$MINIO_DATA_DIR" ] && [ -d "$MINIO_DATA_DIR" ]; then
		rm -rf "$MINIO_DATA_DIR"
	fi
}

cleanup() {
	local status=$?
	stop_minio
	exit "$status"
}

trap cleanup EXIT INT TERM

install_system_ca
ensure_minio

echo "Running integration tests in ${INTEGRATION_TESTPKG}"
echo "Compiling race-enabled test binary (first run can take several minutes with no output)..."
MC_TEST_RUN_FULL_SUITE=true \
MC_TEST_SKIP_BUILD=true \
MC_TEST_BINARY_PATH="${ROOT_DIR}/mc" \
CGO_ENABLED=1 \
go test ${RACE_TEST_FLAGS} -v "${INTEGRATION_TESTPKG}" -run Test_FullSuite

run_functional_tests
