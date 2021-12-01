#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

__cur="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__root="$(cd "$(dirname "${__cur}")" && pwd)"

CADENCE_CLI_DOCKER_IMAGE=ubercadence/cli:0.21.3

docker run -it \
	--network=host --rm \
	--env CADENCE_CLI_ADDRESS=127.0.0.1:7400 \
	--env CADENCE_CLI_DOMAIN=enduro \
		${CADENCE_CLI_DOCKER_IMAGE} $@
