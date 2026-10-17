#!/usr/bin/env bash
# The cloud session's environment check (spc-2610120941441282,
# itd-2610120738075547 C2 and decision 5): before any work, name every tool the
# machine lacks, what supplies it and which preflight gates need it.
#
# It reads scripts/gate-tools.tsv (tool, gates, setup step; the format is
# documented there) and tests each tool with `command -v`. `go` is also held to
# the toolchain go.mod declares: scripts/pinned-toolchain.sh has no check-only
# mode (it resolves by FETCHING the declared toolchain, and a check must change
# nothing), so this compares instead, without a fetch: the go on PATH is the
# declared release, or that release is already in go's toolchain cache, where
# the resolver would have put it.
#
#   exit 0  every tool a gate needs is present (a missing tool no gate needs is
#           a warning line, and the session carries on)
#   exit 1  a tool a preflight gate needs is missing: the session stops before
#           any work and hands back a report saying why (C6), because its
#           checks would fail; also when run as root
#   exit 2  the list is missing or malformed
#
# It writes nothing anywhere: no file, no git command, no cache.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"
LIST="$root/scripts/gate-tools.tsv"

# Every check runs as the normal user the setup created, never as root
# (itd-2610120738075547 decision 8): the gates include tests that prove abcd
# refuses a path other accounts can write, and root passes those vacuously.
if [ "$(id -u)" = 0 ]; then
	echo "env-check: REFUSING to run as root (uid 0); the checks run as the normal user the setup created." >&2
	exit 1
fi

if [ ! -r "$LIST" ]; then
	echo "env-check: cannot read the gate-tool list scripts/gate-tools.tsv" >&2
	exit 2
fi

# toolchain_ready says whether the go on PATH can run the toolchain go.mod
# declares without fetching it.
toolchain_ready() {
	local want="$1" have cache goos goarch
	have="$(GOTOOLCHAIN=local go env GOVERSION 2>/dev/null)" || return 1
	[ "$have" != "go$want" ] || return 0
	cache="$(GOTOOLCHAIN=local go env GOMODCACHE 2>/dev/null)" || return 1
	goos="$(GOTOOLCHAIN=local go env GOOS 2>/dev/null)" || return 1
	goarch="$(GOTOOLCHAIN=local go env GOARCH 2>/dev/null)" || return 1
	[ -n "$cache" ] && [ -x "$cache/golang.org/toolchain@v0.0.1-go$want.$goos-$goarch/bin/go" ]
}

stop=0
missing=0
n=0
while IFS= read -r line || [ -n "$line" ]; do
	n=$((n + 1))
	line="${line%$'\r'}"
	case "$line" in
	'' | '#'*) continue ;;
	esac
	IFS=$'\t' read -r tool gates setup extra <<<"$line" || true
	if [ -z "$tool" ] || [ -z "$gates" ] || [ -z "$setup" ] || [ -n "${extra:-}" ]; then
		echo "env-check: scripts/gate-tools.tsv line $n is not three TAB-separated fields (tool, gates, setup step)" >&2
		exit 2
	fi
	what="$tool"
	present=1
	if ! command -v "$tool" >/dev/null 2>&1; then
		present=0
	elif [ "$tool" = go ]; then
		want="$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' "$root/go.mod" 2>/dev/null | head -n1)"
		if [ -z "$want" ]; then
			echo "env-check: go.mod declares no \`go <version>\` line, so the toolchain cannot be checked" >&2
			exit 2
		fi
		if ! toolchain_ready "$want"; then
			present=0
			what="go$want (the toolchain go.mod declares; the go on PATH is not it and it is not cached)"
		fi
	fi
	[ "$present" -eq 0 ] || continue
	missing=$((missing + 1))
	if [ "$gates" = - ]; then
		echo "env-check: warning: $what is missing; supplied by $setup; no preflight gate needs it, so the session carries on"
	else
		stop=1
		echo "env-check: MISSING $what; supplied by $setup; needed by the preflight gates: $(printf '%s' "$gates" | sed 's/,/, /g')"
	fi
done <"$LIST"

if [ "$stop" -ne 0 ]; then
	echo "env-check: STOPPING before any work: a tool a preflight gate needs is missing, so the hand-back's checks would fail. Hand back a report saying why." >&2
	exit 1
fi
if [ "$missing" -eq 0 ]; then
	echo "env-check: every listed tool is present"
else
	echo "env-check: every tool a preflight gate needs is present"
fi
