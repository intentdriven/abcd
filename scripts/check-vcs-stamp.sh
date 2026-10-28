#!/usr/bin/env bash
# iss-232: the `ahoy install` staleness refusal keys on the binary's BuildInfo
# vcs.revision (internal/core/vintage, current-from-BuildInfo). Release
# binaries are stamped today only by default (-buildvcs=auto from a git
# checkout); a build with -buildvcs=false, or from a .git-less source
# archive, ships binaries whose vintage is undeterminable — Known flips
# false for EVERY pinned user and `ahoy install` starts refusing until
# --allow-stale-binary. This guard makes that coupling impossible to break
# silently: every binary about to be checksummed and published must carry a
# vcs.revision, or the release aborts before anything is hashed or uploaded.
#
# With TAG set (release.yml sets it for the whole run), each binary must also
# claim that release: the main-module version Go stamps from the tag the build
# was made at. abcd update, ahoy install and the plugin bootstrap refuse a
# release binary that claims any other release (ahoy.CheckReleaseClaim), so a
# release whose binaries claimed none would be refused by every client.
set -euo pipefail
shopt -s nullglob

bins=(bin/abcd-*)
if [ ${#bins[@]} -eq 0 ]; then
  echo "check-vcs-stamp: no binaries under bin/ — run \`make build\` first" >&2
  exit 1
fi

fail=0
for b in "${bins[@]}"; do
  info="$(go version -m "$b")"
  if ! printf '%s\n' "$info" | grep -q 'vcs\.revision='; then
    echo "check-vcs-stamp: FAIL $b carries no vcs.revision — pinned users' \`ahoy install\` would refuse (iss-232)" >&2
    fail=1
  elif printf '%s\n' "$info" | grep -q 'vcs\.modified=true'; then
    echo "check-vcs-stamp: FAIL $b was built from a dirty tree (vcs.modified=true) — its vintage is undeterminable and pinned users' \`ahoy install\` would refuse (iss-232)" >&2
    fail=1
  elif [ -n "${TAG:-}" ] && [ "$(printf '%s\n' "$info" | awk '$1 == "mod" && $2 == "github.com/intentdriven/abcd" { print $3 }')" != "$TAG" ]; then
    echo "check-vcs-stamp: FAIL $b does not claim release $TAG in its build metadata — every client would refuse it as not this release's build" >&2
    fail=1
  else
    echo "check-vcs-stamp: OK   $b"
  fi
done
exit "$fail"
