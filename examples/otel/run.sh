#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
collector_image="${OTEL_COLLECTOR_IMAGE:-otel/opentelemetry-collector:0.154.0}"
container_name="gosoline-otel-demo-collector-$$"

prefix_output() {
	local source="$1"
	while IFS= read -r line || [[ -n "$line" ]]; do
		printf '[%s] %s\n' "$source" "$line"
	done
}

if ! command -v docker >/dev/null 2>&1; then
	echo "docker is required to run the OpenTelemetry Collector" >&2
	exit 1
fi

cleanup() {
	if docker container inspect "$container_name" >/dev/null 2>&1; then
		docker stop "$container_name" >/dev/null 2>&1 || true
	fi
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

echo "starting OpenTelemetry Collector ($collector_image)"
docker run --rm \
	--name "$container_name" \
	--publish 4317:4317 \
	--volume "$script_dir/collector-config.yaml:/etc/otelcol/config.yaml:ro" \
	"$collector_image" \
	--config=/etc/otelcol/config.yaml \
	> >(prefix_output collector) 2>&1 &
collector_pid=$!

sleep 1
if ! kill -0 "$collector_pid" 2>/dev/null; then
	wait "$collector_pid"
fi

echo "starting gosoline OTEL metric demo"
(
	cd "$script_dir"
	go run . > >(prefix_output application) 2>&1
)
