#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

curdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd ${curdir}/..

#
# List all packages relevant to coverage reporting.
#
# Usage example:
#
#  $ go test -race -coverprofile=covreport -covermode=atomic -coverpkg=$(hack/coverpkgs.sh) -v ./...
#  $ go tool cover -func=html
#

go list ./... \
	| grep -v "/artefactual-labs/enduro/hack" \
	| grep -v "/artefactual-labs/enduro/internal/api/gen" \
	| grep -v "/artefactual-labs/enduro/internal/api/design" \
	| grep -v "/artefactual-labs/enduro/ui" \
	| grep -v "/fake" \
	| paste -sd","
