#!/usr/bin/env bash
# The one resolver for the Go toolchain go.mod declares (iss-2609081953452204).
# Prints the declared toolchain's GOROOT on stdout and nothing else; every
# diagnostic goes to stderr.
#
#   scripts/pinned-toolchain.sh <version>     # e.g. 1.26.7, read from go.mod by the caller
#
# `GOTOOLCHAIN=go<version> go env GOROOT` fetches and caches the declared
# toolchain if the machine lacks it, then reports where it landed. The format
# gate runs the gofmt under that root, which is the one CI's setup-go installs
# from go.mod.
#
# It REFUSES (exit 2) rather than falling back when the toolchain cannot be
# resolved (offline, or the fetch declined). A fallback would judge the tree
# with a toolchain CI does not use, which is the false green this resolver
# exists to remove, so the refusal names the skew instead
# (.abcd/development/principles/loud-staging.md).
set -euo pipefail

version="${1:-}"
if [ -z "$version" ]; then
	echo "pinned-toolchain: REFUSING — go.mod declares no \`go <version>\` line, so there is no toolchain to resolve." >&2
	exit 2
fi

local_version="$(GOTOOLCHAIN=local go env GOVERSION 2>/dev/null || echo unknown)"

# stdout ALONE is the root. The fetch prints its progress ("go: downloading
# go1.26.7 ...") and any error on stderr, which passes straight through to the
# caller's terminal: merged into the captured value, a first-run progress line
# made the root two lines, and the gate refused on the run where the fetch had
# just succeeded, blaming the network (iss-2609090951287096).
if ! goroot="$(GOTOOLCHAIN="go$version" go env GOROOT)"; then
	echo "pinned-toolchain: REFUSING to judge this tree." >&2
	echo "pinned-toolchain:   go.mod declares go$version; the go on PATH is $local_version." >&2
	echo "pinned-toolchain:   the go$version toolchain could not be fetched (go's own error is above; the fetch needs network)." >&2
	echo "pinned-toolchain:   NOT falling back to the go on PATH — a different toolchain judges this" >&2
	echo "pinned-toolchain:   tree differently, so the fallback would pass what CI refuses." >&2
	exit 2
fi
if [ ! -x "$goroot/bin/gofmt" ] || [ ! -x "$goroot/bin/go" ]; then
	echo "pinned-toolchain: REFUSING — go$version resolved to '$goroot', which holds no bin/go and bin/gofmt." >&2
	echo "pinned-toolchain:   go.mod declares go$version; the go on PATH is $local_version." >&2
	exit 2
fi

resolved="$("$goroot/bin/go" version 2>/dev/null | awk '{print $3}')"
if [ "$resolved" != "go$version" ]; then
	echo "pinned-toolchain: REFUSING — go.mod declares go$version, but the resolved toolchain reports $resolved." >&2
	echo "pinned-toolchain:   GOTOOLCHAIN did not switch, so the gate would run the wrong toolchain." >&2
	exit 2
fi

printf '%s\n' "$goroot"
