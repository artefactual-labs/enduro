#!/usr/bin/env bash

fail() {
    printf '%s\n' "$*" >&2
    exit 1
}

read_metadata() {
    version=$(jq -er '.version | select(type == "string" and length > 0)' "$1")
    commit=$(jq -er '.commit | select(type == "string" and test("^[0-9a-f]{40}$"))' "$1")
}

image_tag() {
    printf 'ghcr.io/artefactual-labs/enduro:%s-%s\n' "$version" "$1"
}

archive_checksum() {
    shasum -a 256 "$1" | awk '{print $1}'
}

check_image() {
    local image=$1 arch=$2
    docker image inspect "$image" | jq -e --arg arch "$arch" --arg version "$version" --arg commit "$commit" '
        length == 1 and (.[0] |
            .Os == "linux" and .Architecture == $arch and
            .Config.User == "65532:65532" and
            .Config.WorkingDir == "/home/nonroot" and
            .Config.Entrypoint == ["/usr/local/bin/enduro"] and
            .Config.Cmd == ["--config", "/etc/enduro/enduro.toml"] and
            (.Config.Env | index("HOME=/home/nonroot") != null) and
            .Config.Labels["org.opencontainers.image.version"] == $version and
            .Config.Labels["org.opencontainers.image.revision"] == $commit)
    ' >/dev/null || fail "Unexpected image configuration: $image"
}
