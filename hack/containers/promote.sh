#!/usr/bin/env bash
set -euo pipefail

# Resolve a candidate once, then pass its immutable reference to tests and promotion.
# Promotion copies the complete index, including BuildKit attestation manifests.
usage() {
    echo "Usage: $0 --resolve IMAGE:TAG | IMAGE@sha256:DIGEST IMAGE:VERSION" >&2
    exit 2
}
[[ $# == 2 ]] || usage
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

inspect() {
    local attempt
    for attempt in 1 2 3; do
        if docker buildx imagetools inspect "$1" > "$work/inspect" 2> "$work/error"; then
            awk '$1 == "Digest:" {print $2; exit}' "$work/inspect"
            return
        fi
        # A missing tag is different from an authentication or transport failure.
        if grep -Eq 'manifest unknown|: not found' "$work/error"; then
            return 4
        fi
        sleep "$attempt"
    done
    cat "$work/error" >&2
    return 1
}

validate_index() {
    docker buildx imagetools inspect --raw "$1" > "$work/index.json"
    jq -e '
        (.mediaType == "application/vnd.oci.image.index.v1+json" or
         .mediaType == "application/vnd.docker.distribution.manifest.list.v2+json") and
        ([.manifests[] | select(.platform.os != "unknown") |
          "\(.platform.os)/\(.platform.architecture)"] | sort == ["linux/amd64", "linux/arm64"]) and
        all(.manifests[]; .platform.os != "unknown" or
          .annotations["vnd.docker.reference.type"] == "attestation-manifest")
    ' "$work/index.json" > /dev/null
}

if [[ $1 == --resolve ]]; then
    candidate=$2
    digest=$(inspect "$candidate")
    [[ $digest =~ ^sha256:[a-f0-9]{64}$ ]] || exit 1
    reference="${candidate%:*}@$digest"
    validate_index "$reference"
    printf '%s\n' "$reference"
    exit
fi

source=$1
target=$2
[[ $source =~ @sha256:[a-f0-9]{64}$ ]] || usage
[[ $target == "${source%@*}:"* && $target != *@* ]] || usage
validate_index "$source"
expected=${source##*@}
if actual=$(inspect "$target"); then
    if [[ $actual != "$expected" ]]; then
        echo "Refusing to replace $target: $actual differs from tested $expected" >&2
        exit 1
    fi
    echo "$target already points to tested $expected"
    exit
else
    status=$?
    [[ $status == 4 ]] || exit "$status"
fi

# A single index source performs a carbon copy; do not filter platforms or add
# annotations here, since either would change the tested index digest.
docker buildx imagetools create --tag "$target" "$source"
actual=$(inspect "$target")
[[ $actual == "$expected" ]] || {
    echo "Promoted digest $actual differs from tested $expected" >&2
    exit 1
}
echo "Promoted $target at $expected"
