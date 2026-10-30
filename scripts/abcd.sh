#!/usr/bin/env bash
# The source checkout's own abcd (iss-2610241240133009): the tree's binary,
# built to the checkout's local tier and run there, so its exit code reaches
# the caller. `go run ./cmd/abcd` turned every nonzero exit into 1, so exit 2
# (a usage error, a guard that could not judge) and exit 3 (reserve: back off)
# arrived as the 1 a guard block also exits with.
#
# `go build -o` relinks only when an input changed: it compares the build ID it
# would write with the one the existing binary carries, so an unchanged tree
# costs a hash of its inputs and no compile. The build goes to a private name
# that starts as a hard link to the current binary, then is renamed into place,
# so a call running at the same moment runs the old binary or the new one,
# never one half written, even where go copies its output across file systems.
#
# -buildvcs=false builds what `go run` built: no VCS stamp, so no `git status`
# on every call, and one binary whether the tree is the main checkout or a
# linked worktree (where go finds no .git directory to stamp from). The path is
# abcdrepo.CheckoutBuildRel, and the script names the binary it has just built
# in ABCD_CHECKOUT_BUILD (abcdrepo.CheckoutBuildEnv) before it runs it, which is
# how vintage.CheckoutBuild tells the tree as it stands from a copy left in the
# tier and run directly. In a checkout this account cannot write to, the build
# goes to a private temporary directory instead, runs as a child, and is
# removed afterwards.
#
# Usage: scripts/abcd.sh <verb> [args...] from the checkout root, or by its path
# (or a symlink to it) from anywhere; the verb runs in the caller's working
# directory. A tree that does not build exits 2 and runs nothing, so a broken
# build never reads as a guard's block.
set -euo pipefail
# The mark below vouches for the build this run makes and starts, and for
# nothing an earlier run left in the environment.
unset ABCD_CHECKOUT_BUILD

src="${BASH_SOURCE[0]}"
while [ -L "$src" ]; do
	dir="$(CDPATH='' cd -P -- "$(dirname -- "$src")" && pwd)"
	src="$(readlink -- "$src")"
	case $src in /*) ;; *) src="$dir/$src" ;; esac
done
root="$(CDPATH='' cd -P -- "$(dirname -- "$src")/.." && pwd)"
if [ ! -f "$root/cmd/abcd/main.go" ]; then
	# The folder in its ~ form, as every abcd surface prints a home path. The
	# literal ~ is the output wanted, not an expansion left undone.
	shown="$root"
	# shellcheck disable=SC2088
	case $root in "${HOME:-/nonexistent}"/*) shown="~/${root#"$HOME"/}" ;; esac
	echo "scripts/abcd.sh: $shown holds no cmd/abcd/main.go, so it is not a source checkout of abcd; nothing ran" >&2
	exit 2
fi

# build <output>: link the tree to <output>, or exit 2 having run nothing.
build() {
	if ! (cd "$root" && go build -buildvcs=false -o "$1" ./cmd/abcd); then
		echo "scripts/abcd.sh: cmd/abcd did not build, so nothing ran" >&2
		exit 2
	fi
}

bin="$root/.abcd/.work.local/bin/abcd"
if ! mkdir -p -- "${bin%/*}" 2>/dev/null || [ ! -w "${bin%/*}" ]; then
	tmp="$(mktemp -d "${TMPDIR:-/tmp}/abcd.XXXXXX")"
	trap 'rm -rf -- "$tmp"' EXIT
	build "$tmp/abcd"
	"$tmp/abcd" "$@" && rc=0 || rc=$?
	exit "$rc"
fi

next="$bin.$$"
trap 'rm -f -- "$next"' EXIT
ln -f -- "$bin" "$next" 2>/dev/null || rm -f -- "$next"
build "$next"
if [ "$next" -ef "$bin" ]; then
	rm -f -- "$next"
else
	mv -f -- "$next" "$bin"
fi
trap - EXIT
export ABCD_CHECKOUT_BUILD="$bin"
exec "$bin" "$@"
