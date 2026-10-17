#!/usr/bin/env bash
# Print the name of origin's default branch (spc-2610120941441282), the branch
# a cloud brief's item must already be on and a hosted session starts from.
#
# Read from the local origin/HEAD when it is set, otherwise from the remote's
# own symbolic HEAD (`git ls-remote --symref origin HEAD`). It never guesses:
# no `main` fallback, because a wrong answer here reads a private record as
# public (C13). Exit 0 with the name on standard output; exit 1, naming the
# cause, when it cannot be determined.
#
# Usage: default-branch.sh   (run from inside the checkout)
set -euo pipefail

if [ $# -ne 0 ]; then
	echo "usage: default-branch.sh" >&2
	exit 2
fi

name=""
if ref="$(git symbolic-ref -q refs/remotes/origin/HEAD 2>/dev/null)"; then
	name="${ref#refs/remotes/origin/}"
else
	# "ref: refs/heads/<name>	HEAD" is the line naming the symbolic ref.
	if ! listing="$(git ls-remote --symref origin HEAD 2>/dev/null)"; then
		echo "default-branch: cannot read origin's default branch: the remote did not answer." >&2
		exit 1
	fi
	name="$(printf '%s\n' "$listing" | sed -n 's|^ref: refs/heads/\([^	]*\)	HEAD$|\1|p' | head -n 1)"
fi

if [ -z "$name" ] || ! git check-ref-format --branch "$name" >/dev/null 2>&1; then
	echo "default-branch: cannot determine origin's default branch (origin/HEAD is unset and the remote names none)." >&2
	exit 1
fi
printf '%s\n' "$name"
