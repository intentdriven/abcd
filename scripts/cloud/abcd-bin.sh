#!/usr/bin/env bash
# Print the path of an abcd binary built from this checkout, for the cloud
# scripts that read abcd's answers (brief.sh, hand-back.sh, list.sh).
#
# They branch on abcd's exit codes (2 is the embargo check's refusal and a
# verb that cannot run; 1 a gate's refusal), and `go run` turns every non-zero
# exit of the program it runs into its own exit 1. So a script that runs abcd
# through `go run` can never tell a refusal from a check that could not run.
# A built binary's exit code is abcd's own.
#
#   - ABCD_BIN is printed as it is only when it names the binary the cloud
#     setup builds in this checkout, .abcd/.work.local/cloud/bin/abcd (built
#     per session start and exported, connectors/<host>/setup.sh): a regular,
#     executable file, not a symlink, whose folder, resolved with pwd -P, is
#     this checkout's .abcd/.work.local/cloud/bin, resolved the same way.
#     Anything else (an installed abcd, which is whatever was last released;
#     another checkout's build; a variable left set on another machine) is
#     ignored, with a note, and abcd is built.
#   - Otherwise abcd is built, once, with `go build -o <dir>/abcd ./cmd/abcd`
#     from the checkout's root, and that path is printed. <dir> is the
#     caller's own temporary directory, which its exit trap removes.
#
# Usage: abcd-bin.sh <dir>
# Exit 0 the path on standard output; 1 the build failed (the build's own
# output is on standard error); 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

if [ $# -ne 1 ] || [ -z "$1" ] || [ ! -d "$1" ]; then
	echo "usage: abcd-bin.sh <dir>   (an existing directory the binary is built into)" >&2
	exit 2
fi

# setup_built <path> succeeds when path is the setup's binary in this checkout.
setup_built() {
	local want got
	[ -f "$1" ] && [ ! -L "$1" ] && [ -x "$1" ] && [ "$(basename -- "$1")" = abcd ] || return 1
	want="$(cd "$root/.abcd/.work.local/cloud/bin" 2>/dev/null && pwd -P)" || return 1
	got="$(cd "$(dirname -- "$1")" 2>/dev/null && pwd -P)" || return 1
	[ -n "$want" ] && [ "$want" = "$got" ]
}

if [ -n "${ABCD_BIN:-}" ]; then
	if setup_built "$ABCD_BIN"; then
		printf '%s\n' "$ABCD_BIN"
		exit 0
	fi
	echo "abcd-bin: ABCD_BIN names no binary the cloud setup built in this checkout (.abcd/.work.local/cloud/bin/abcd), so abcd is built from the checkout instead." >&2
fi

out="$1/abcd"
cd "$root" || exit 1
if ! go build -o "$out" ./cmd/abcd >&2 </dev/null; then
	exit 1
fi
[ -x "$out" ] || exit 1
printf '%s\n' "$out"
