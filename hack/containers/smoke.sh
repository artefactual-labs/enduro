#!/usr/bin/env bash
# shellcheck source-path=SCRIPTDIR
set -euo pipefail

source "$(dirname "$0")/common.sh"
image_ref=
if [[ ${1:-} == --* ]]; then
    arch_selection=all
    version=
    commit=
    while [[ $# -gt 0 ]]; do
        [[ $# -ge 2 ]] || fail "Missing value for $1"
        case $1 in
            --image-ref) image_ref=$2 ;;
            --arch) arch_selection=$2 ;;
            --commit) commit=$2 ;;
            --version) version=$2 ;;
            *) fail "Unknown option: $1" ;;
        esac
        shift 2
    done
    [[ -n "$image_ref" && -n "$version" && "$commit" =~ ^[0-9a-f]{40}$ ]] || \
        fail "Registry checks require --image-ref, --arch, --version and --commit"
    [[ "$arch_selection" != all ]] || fail "Registry checks require one native --arch"
    [[ "$image_ref" =~ @sha256:[0-9a-f]{64}$ ]] || fail "Registry image must use an immutable digest"
    artifacts=
else
    [[ $# -le 2 ]] || fail "Usage: $0 [amd64|arm64|all] [artifact-directory]"
    arch_selection=${1:-all}
    artifacts=${2:-}
    read_metadata "${artifacts:-dist}/metadata.json"
fi
case $arch_selection in
    amd64|arm64) arches=("$arch_selection") ;;
    all) arches=(amd64 arm64) ;;
    *) fail "Unsupported architecture: $arch_selection" ;;
esac
for arch in "${arches[@]}"; do
    image=${image_ref:-$(image_tag "$arch")}
    if [[ -n "$image_ref" ]]; then
        docker pull --platform "linux/$arch" "$image"
    fi
    if [[ -n "$artifacts" ]]; then
        # Fixed filenames avoid letting metadata select arbitrary local paths.
        archive="$artifacts/enduro-$arch.tar"
        jq -e --arg arch "$arch" --arg image "$image" \
            '.images[$arch] | .image == $image and .archive == ("enduro-" + $arch + ".tar")' \
            "$artifacts/metadata.json" >/dev/null || fail "Unexpected archive identity for $arch"
        checksum=$(jq -er --arg arch "$arch" '.images[$arch].sha256' "$artifacts/metadata.json")
        [[ $(archive_checksum "$archive") == "$checksum" ]] || fail "Archive checksum mismatch: $archive"
        docker image load --input "$archive"
        id=$(jq -er --arg arch "$arch" '.images[$arch].id' "$artifacts/metadata.json")
        [[ $(docker image inspect --format '{{.Id}}' "$image") == "$id" ]] || fail "Image identity mismatch: $image"
    fi
    check_image "$image" "$arch"
    output=$(docker run --rm --platform "linux/$arch" --network none \
        --read-only --cap-drop ALL --security-opt no-new-privileges \
        "$image" --version)
    printf '%s\n' "$output"
    [[ "$output" == "enduro version $version (commit=$commit) "* ]]
    # The production default must reach the expected config path and fail
    # promptly when no configuration has been mounted.
    if output=$(docker run --rm --platform "linux/$arch" --network none \
        --read-only --cap-drop ALL --security-opt no-new-privileges "$image" 2>&1); then
        fail "Image started without its required configuration: $image"
    fi
    [[ "$output" == *"Failed to read configuration:"*"/etc/enduro/enduro.toml"* ]]
done
