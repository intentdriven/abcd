#!/usr/bin/env bash
# Compose the cloud brief for one item (spc-2610120941441282 step 5,
# itd-2610120738075547 C4 and C13, decisions 7, 14 and 17). Run on the
# person's own computer, by send.sh or alone to preview; the brief goes to
# standard output and nowhere else.
#
# It refuses, printing no brief, unless in order:
#
#   1. the item is ready to build: an open issue record under
#      .abcd/work/issues/open/, or a planned intent for which
#      `go run ./cmd/abcd intent ready <id> --json` exits 0 (decision 7);
#   2. the item is public: after `git fetch origin`, its record's path exists
#      in the tree of origin's default branch (`git cat-file -e`). A draft never
#      pushed, a record only on another branch and a private record all fail
#      here, before any brief is composed (C13);
#   3. the composed brief holds no path on this computer (its home, the
#      checkout), no session link and no secret shape, and passes
#      `go run ./cmd/abcd lint outbound --label brief` (C4);
#   4. the item is not under embargo: when the embargo store
#      (~/.abcd.noindex/worktrees/<root-sha>/embargo/) exists on this machine,
#      the composed brief passes `go run ./cmd/abcd embargo check --file`. A
#      store that exists while that check cannot run is a refusal, never a pass
#      (fail closed). With no store, nothing is under embargo here.
#
# The brief names no author. It states the rule that every commit is authored
# under the identity the setup took from the forge token (decision 17), and
# this script never asks git, or anything else here, whose computer it runs on.
#
# Every host-specific value (the work-in-progress prefix) comes from the
# connector (scripts/cloud/connector.sh), so nothing here names a host (C12).
#
# Usage: brief.sh [--connector <name>] <item-id>
#   (default connector: the only one declared)
# Exit 0 with the brief on standard output; 1 refused, the reason on standard
# error; 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

ITEM_RE='^(iss|itd)-([0-9]{16}|[0-9]{1,4})$'
# Every value the template is filled with is held to this, so it can go
# through sed with no escaping and carry nothing but a path or a name.
SAFE_RE='^[A-Za-z0-9._/-]+$'

usage() {
	echo "usage: brief.sh [--connector <name>] <item-id>   (an iss- or itd- record id)" >&2
	exit 2
}

refuse() {
	echo "brief: REFUSING $1" >&2
	exit 1
}

connector=""
item=""
while [ $# -gt 0 ]; do
	case "$1" in
	--connector)
		[ $# -ge 2 ] || usage
		connector="$2"
		shift 2
		;;
	-*) usage ;;
	*)
		[ -z "$item" ] || usage
		item="$1"
		shift
		;;
	esac
done
[[ "$item" =~ $ITEM_RE ]] || usage

cd "$root"

if [ -z "$connector" ]; then
	connector="$(bash "$here/connector.sh" default)" || exit 1
fi
prefix="$(bash "$here/connector.sh" get "$connector" wip_prefix)" || exit 1
branch="$prefix$item"

# --- 1. Ready to build ------------------------------------------------------

case "$item" in
iss-*)
	dir=".abcd/work/issues/open"
	kind="an open issue record"
	;;
itd-*)
	dir=".abcd/development/intents/planned"
	kind="a planned intent"
	;;
esac
record=""
n=0
for f in "$dir/$item"-*.md; do
	if [ -f "$f" ] && [ ! -L "$f" ]; then
		n=$((n + 1))
		record="$f"
	fi
done
if [ "$n" -eq 0 ]; then
	refuse "$item: it is not ready to build: there is no $kind for it under $dir/ (an item ready to build is an open issue or a planned intent)."
fi
[ "$n" -eq 1 ] || refuse "$item: $n records under $dir/ carry this id; fix the ledger first."
[[ "$record" =~ $SAFE_RE ]] || refuse "$item: its record's file name $record holds characters a brief does not carry."

if [ "$kind" = "a planned intent" ]; then
	rc=0
	gate="$(go run ./cmd/abcd intent ready "$item" --json 2>&1)" || rc=$?
	case "$rc" in
	0) ;;
	1)
		printf '%s\n' "$gate" >&2
		refuse "$item: it is not ready: its readiness gate (go run ./cmd/abcd intent ready) does not pass; the checks are above."
		;;
	*)
		printf '%s\n' "$gate" >&2
		refuse "$item: its readiness gate (go run ./cmd/abcd intent ready) could not run (exit $rc), so it is not known to be ready."
		;;
	esac
fi

# --- 2. Public ---------------------------------------------------------------

git fetch -q origin 2>/dev/null ||
	refuse "$item: cannot fetch origin, so whether it is public cannot be read."
base="$(bash "$here/default-branch.sh")" ||
	refuse "$item: origin's default branch cannot be determined, so whether it is public cannot be read."
[[ "$base" =~ $SAFE_RE ]] || refuse "$item: origin's default branch has a name a brief does not carry."
git rev-parse -q --verify "refs/remotes/origin/$base^{commit}" >/dev/null ||
	refuse "$item: origin's default branch, $base, was not fetched, so whether it is public cannot be read."
git cat-file -e "refs/remotes/origin/$base:$record" 2>/dev/null ||
	refuse "$item: it is not public yet: its record $record is not on origin's default branch ($base). Push it there first; a cloud session's work is public, so nothing private is handed to one."

# --- 3. Compose, and check what was composed ---------------------------------

# report_dir from cloud.conf, read as data and never sourced.
report_dir="$(sed -n 's/^report_dir=//p' "$here/cloud.conf" | tail -n 1)"
report_dir="${report_dir%/}"
case "/$report_dir/" in
*/../* | */./* | //*) report_dir="" ;;
esac
[[ "$report_dir" =~ $SAFE_RE ]] ||
	refuse "scripts/cloud/cloud.conf: report_dir must be a relative path inside the checkout."

tmp="$(mktemp "${TMPDIR:-/tmp}/abcd-brief.XXXXXX")"
trap 'rm -f -- "$tmp"' EXIT

sed -e "s|{{ITEM}}|$item|g" -e "s|{{RECORD}}|$record|g" -e "s|{{BASE}}|$base|g" \
	-e "s|{{BRANCH}}|$branch|g" -e "s|{{REPORT_DIR}}|$report_dir|g" \
	"$here/brief.md.tmpl" >"$tmp"
if grep -qE '\{\{|\}\}' "$tmp"; then
	refuse "scripts/cloud/brief.md.tmpl carries a placeholder this script does not fill."
fi

# found <what> <grep -E pattern>: refuse when the brief matches, naming the
# line and never repeating what matched.
found() {
	local what="$1" re="$2" lines
	lines="$(grep -nE -- "$re" "$tmp" | cut -d: -f1 | paste -sd, - || true)"
	[ -z "$lines" ] || refuse "the brief for $item carries $what (line $lines); a brief leaves this computer, so it carries none."
}
contains() {
	local what="$1" text="$2" lines
	lines="$(grep -nF -- "$text" "$tmp" | cut -d: -f1 | paste -sd, - || true)"
	[ -z "$lines" ] || refuse "the brief for $item carries $what (line $lines); a brief leaves this computer, so it carries none."
}
if [ -n "${HOME:-}" ] && [ "$HOME" != / ]; then
	contains "a path in this computer's home folder" "$HOME"
fi
contains "a path to this checkout on this computer" "$root"
found "a home path" '(^|[^A-Za-z0-9_.-])/(Users|home)/[A-Za-z0-9._-]+'
found "a session link" 'https?://[^[:space:]]*session[_/=-]'
found "a secret" 'gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----|xox[abeprs]-[A-Za-z0-9-]{10,}|sk-[A-Za-z0-9_-]{20,}'

rc=0
lint="$(go run ./cmd/abcd lint outbound --label brief "$tmp" 2>&1)" || rc=$?
case "$rc" in
0) ;;
1)
	printf '%s\n' "$lint" >&2
	refuse "the brief for $item fails the outbound check (go run ./cmd/abcd lint outbound); the finding is above."
	;;
*)
	printf '%s\n' "$lint" >&2
	refuse "the outbound check (go run ./cmd/abcd lint outbound) could not run (exit $rc), so the brief for $item is unchecked."
	;;
esac

# --- 4. Not under embargo ----------------------------------------------------

[ -n "${HOME:-}" ] ||
	refuse "HOME is unset, so whether an embargo store exists on this machine cannot be read."
root_sha="$(git rev-list -n 1 --max-parents=0 HEAD 2>/dev/null)" || root_sha=""
[[ "$root_sha" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]] ||
	refuse "this repository's root commit cannot be read, so whether an embargo store exists on this machine cannot be read."
store="$HOME/.abcd.noindex/worktrees/$root_sha/embargo"
if [ -e "$store" ] || [ -L "$store" ]; then
	rc=0
	check="$(go run ./cmd/abcd embargo check --file "$tmp" 2>&1)" || rc=$?
	case "$rc" in
	0) ;;
	2)
		printf '%s\n' "$check" >&2
		refuse "$item: it is held under embargo: the embargo check refuses its brief (go run ./cmd/abcd embargo check); the finding is above."
		;;
	*)
		printf '%s\n' "$check" >&2
		refuse "$item: an embargo store exists on this machine and its check (go run ./cmd/abcd embargo check) could not run (exit $rc); an unchecked brief is never passed."
		;;
	esac
fi

cat "$tmp"
