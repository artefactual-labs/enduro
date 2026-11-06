#!/usr/bin/env bash
# Export smoke diagnostics before propagating the test result to the caller.
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 SUITE ARTIFACT_DIRECTORY [DAGGER_ARGUMENTS...]" >&2
  exit 2
fi

suite=$1
artifacts=$2
shift 2
case "$suite" in
  smoke-tests|object-storage-smoke-test)
    results=(smoke-exit-code)
    ;;
  object-storage-smoke-tests)
    results=(minio-latest/smoke-exit-code silo/smoke-exit-code seaweedfs/smoke-exit-code argmin/smoke-exit-code)
    ;;
  coverage-check|startup-test)
    results=()
    ;;
  *)
    echo "Unknown suite: $suite" >&2
    exit 2
    ;;
esac

cd "$(dirname "$0")/../.."
dagger --progress=plain -m hack/dagger api call --report-only "$suite" \
  --source . "$@" export --path "$artifacts" --wipe

# Bash 3.2 treats an empty array as unset under nounset.
if [[ "$suite" == coverage-check || "$suite" == startup-test ]]; then
  exit 0
fi

failed=0
for result in "${results[@]}"; do
  # A missing or malformed result is a failure, never a successful test run.
  if [[ ! -f "$artifacts/$result" ]] || [[ $(cat "$artifacts/$result") != 0 ]]; then
    echo "Smoke suite failed; see $artifacts (result: $result)" >&2
    failed=1
  fi
done
exit "$failed"
