#!/usr/bin/env bash
# shellcheck source-path=SCRIPTDIR
set -euo pipefail

# Export the images already built and loaded by GoReleaser; never rebuild here.
source "$(dirname "$0")/common.sh"
[[ $# -le 1 ]] || fail "Usage: $0 [dist-directory]"
dist=${1:-dist}
read_metadata "$dist/metadata.json"
mkdir -p "$dist/containers"
staging=$(mktemp -d "$dist/containers/.export.XXXXXX")
trap 'rm -rf "$staging"' EXIT
jq -n --arg version "$version" --arg commit "$commit" \
    '{version: $version, commit: $commit, images: {}}' >"$staging/metadata.json"
for arch in amd64 arm64; do
    image=$(image_tag "$arch")
    check_image "$image" "$arch"
    id=$(docker image inspect --format '{{.Id}}' "$image")
    archive="enduro-$arch.tar"
    docker image save --output "$staging/$archive" "$image"
    # Dagger imports OCI archives. Docker's containerd image store includes
    # the OCI layout alongside its Docker-compatible manifest in save output.
    tar -tf "$staging/$archive" >"$staging/entries"
    grep -qx 'oci-layout' "$staging/entries" || \
        fail "Docker save did not produce an OCI archive; enable Docker's containerd image store"
    checksum=$(archive_checksum "$staging/$archive")
    jq --arg arch "$arch" --arg image "$image" --arg id "$id" \
        --arg archive "$archive" --arg checksum "$checksum" \
        '.images[$arch] = {image: $image, id: $id, archive: $archive, sha256: $checksum}' \
        "$staging/metadata.json" >"$staging/metadata.next.json"
    mv "$staging/metadata.next.json" "$staging/metadata.json"
done
mv "$staging"/*.tar "$dist/containers/"
mv "$staging/metadata.json" "$dist/containers/metadata.json"
printf 'Exported snapshot images to %s/containers\n' "$dist"
