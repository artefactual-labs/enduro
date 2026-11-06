#!/usr/bin/env bash
# Exercise the real public smoke entry point and host export on a failing test.
set -euo pipefail
cd "$(dirname "$0")/../.."
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/source" "$fixture/artifacts"
# A reused export directory must not retain reports from a previous invocation.
printf 'stale coverage\n' > "$fixture/artifacts/stale.coverprofile"
printf 'previous cleanup failure\n' > "$fixture/artifacts/cleanup-error.txt"
tar --exclude=.git --exclude=node_modules --exclude=./hack/dagger/runtime \
  -cf - . | tar -xf - -C "$fixture/source"
cat > "$fixture/source/hack/dagger/smoke/tests/ambox-smoke.spec.ts" <<'EOF'
import { expect, test } from "@playwright/test";
import { promises as fs } from "node:fs";

test("failure artifacts survive export", async () => {
  await fs.writeFile(`${process.env.ARTIFACTS_DIR}/failure-sentinel.txt`, "retained\n");
  expect("intentional regression fixture").toBe("success");
});
EOF

status=0
hack/dagger/run.sh smoke-tests "$fixture/artifacts" --source "$fixture/source" || status=$?
test "$status" -eq 1
test "$(cat "$fixture/artifacts/smoke-exit-code")" = 1
test "$(cat "$fixture/artifacts/failure-sentinel.txt")" = retained
test ! -e "$fixture/artifacts/stale.coverprofile"
test ! -e "$fixture/artifacts/cleanup-error.txt"
grep -q 'intentional regression fixture' "$fixture/artifacts/playwright.log"
test -n "$(find "$fixture/artifacts/playwright-results" -name trace.zip -print -quit)"
echo "Failed smoke tests replace stale diagnostics and return a failing status."
