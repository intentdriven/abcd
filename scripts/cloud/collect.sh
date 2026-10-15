#!/usr/bin/env bash
# Collect one cloud session's work onto the person's own computer
# (spc-2610120941441282 step 11, itd-2610120738075547 C9, decision 9). Run on
# the person's computer, from their checkout. In order, and each refusal comes
# before anything is made:
#
#   1. fetch origin's default branch and the item's work-in-progress branch,
#      <wip_prefix><item-id> for the connector's prefix (connector.sh, so
#      nothing here names a host, C12);
#   2. refuse unless the branch's last commit is the report commit alone: one
#      non-merge commit changing <report_dir>/<item-id>.md (cloud.conf) and
#      nothing else. Every commit before it, back to where the branch left the
#      default branch, is the work, but for earlier report commits (each
#      changing this item's report alone, one per earlier hand-back), which are
#      left out as the tip is; refuse when there is none, when one is a
#      merge, when one touches <report_dir>/ (a report never lands) or the
#      local tier .abcd/.work.local/ (it never travels: a receipt or a
#      handover carried on a branch is another machine's), or when the
#      attribution gate (scripts/check-attribution.sh commits) refuses them.
#      The commits must already name the person as author: collect never
#      rewrites authorship. Refuse, too, when the work changes the tooling
#      (the build file, scripts/, go.mod, go.sum, and every top-level hidden
#      directory but .abcd/: the hooks, the workflows, a host's settings),
#      printing the diffstat of those paths, unless
#      --allow-tooling-changes is passed: preflight on the collected tree runs
#      that tree's code as the person, so by default the gates that run are
#      the default branch's own;
#   3. make a worktree in the machine store,
#      ~/.abcd.noindex/worktrees/<root-sha>/collect-<item-id>/
#      (adr-2610031751065746; <root-sha> is the full name of the repository's
#      root commit; no level of it may be a symlink), on a new branch
#      collect/<item-id> from origin's default branch, and cherry-pick the
#      work onto it, the report commits left out, with the hooks of the
#      person's own checkout (core.hooksPath by absolute path), never hooks a
#      collected commit put in the worktree;
#   4. run `make preflight` there, through
#      `go run ./cmd/abcd implement load --wait -- make -C <worktree> preflight`,
#      the machine's one waiter, so it waits while another preflight or race
#      run is live. The waiter is run from this checkout, not from the
#      collected tree;
#   5. on a pass, print the branch, the receipt preflight minted in the
#      worktree's local tier and the push command, for the person to run; on a
#      failure, print the failures and where the worktree is kept for
#      inspection.
#
# It pushes nothing, ever. Any receipt the cloud session made is ignored: the
# verdict is this run's preflight alone, and the receipt printed is the one it
# minted here, for the collected commit (decision 9).
#
# Usage: collect.sh <item-id> [--allow-tooling-changes] [--connector <name>]
#   (default connector: the only one declared)
# Exit 0 preflight passed and the branch is ready to push; 1 refused, or
# preflight failed; 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

# One record id: the hook's own shape for what follows a work-in-progress
# prefix (.githooks/pre-push, wip_id_re).
ITEM_RE='^((iss|itd|spc)-[0-9]{16}|(iss|itd)-[0-9]{1,4})$'
SAFE_RE='^[A-Za-z0-9._/-]+$'
LOCAL_TIER=".abcd/.work.local"

usage() {
	echo "usage: collect.sh <item-id> [--allow-tooling-changes] [--connector <name>]" >&2
	exit 2
}

refuse() {
	echo "collect: REFUSING $1 Nothing was made and nothing was pushed." >&2
	exit 1
}

item="" connector="" allow_tooling=""
while [ $# -gt 0 ]; do
	case "$1" in
	--allow-tooling-changes)
		[ -z "$allow_tooling" ] || usage
		allow_tooling=1
		shift
		;;
	--connector)
		[ $# -ge 2 ] && [ -z "$connector" ] || usage
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
[ -n "$item" ] || usage
[[ "$item" =~ $ITEM_RE ]] ||
	refuse "the item is not a record id (iss-, itd- or spc- and sixteen digits, or an ordinal iss- or itd- id)."

cd "$root"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || refuse "this is not a git checkout."

if [ -z "$connector" ]; then
	connector="$(bash "$here/connector.sh" default)" || exit 1
fi
prefix="$(bash "$here/connector.sh" get "$connector" wip_prefix)" || exit 1
wip="$prefix$item"
landing="collect/$item"

# cloud.conf, read as data and never sourced.
report_dir="$(sed -n 's/^report_dir=//p' "$here/cloud.conf" | tail -n 1)"
report_dir="${report_dir%/}"
case "/$report_dir/" in
*/../* | */./* | //*) report_dir="" ;;
esac
[[ "$report_dir" =~ $SAFE_RE ]] ||
	refuse "scripts/cloud/cloud.conf: report_dir must be a relative path inside the checkout."
report="$report_dir/$item.md"

# --- 1. Fetch -----------------------------------------------------------------

default="$(bash "$here/default-branch.sh")" || exit 1
GIT_TERMINAL_PROMPT=0 git fetch -q origin "+refs/heads/$default:refs/remotes/origin/$default" ||
	refuse "origin's default branch, $default, could not be fetched."
if ! GIT_TERMINAL_PROMPT=0 git fetch -q origin "+refs/heads/$wip:refs/remotes/origin/$wip" 2>/dev/null; then
	refuse "origin holds no branch $wip to collect (or it could not be fetched)."
fi
base="refs/remotes/origin/$default"
tip="$(git rev-parse --verify --quiet "refs/remotes/origin/$wip^{commit}")" ||
	refuse "origin holds no branch $wip to collect."

# --- 2. The shape of the branch -------------------------------------------------

# paths <commit> lists what one commit changes against its parent.
paths() {
	git diff-tree --no-commit-id --name-only -r --no-renames "$1"
}

parents="$(git rev-list --parents -n 1 "$tip" | wc -w | tr -d ' ')"
[ "$parents" -eq 2 ] && [ "$(paths "$tip")" = "$report" ] ||
	refuse "the last commit on $wip is not the report commit alone (one commit changing $report and nothing else); the hand-back writes it last."

fork="$(git merge-base "$base" "$tip")" ||
	refuse "$wip shares no history with origin's $default."
[ -z "$(git rev-list --merges "$fork..$tip^")" ] ||
	refuse "$wip holds a merge commit; collect lands a straight line of the session's own commits."
# The work is every commit before the tip but the earlier report commits: a
# session that stopped and was later handed back leaves one report commit per
# hand-back, each changing this item's report and nothing else. Those are left
# out as the tip is; any other commit touching the report folder is refused
# below.
work=""
for sha in $(git rev-list --reverse "$fork..$tip^"); do
	[ "$(paths "$sha")" = "$report" ] && continue
	work="${work:+$work
}$sha"
done
[ -n "$work" ] ||
	refuse "$wip carries the report and no work: there is nothing to collect. Read the report with scripts/cloud/list.sh."
while IFS= read -r sha; do
	while IFS= read -r p; do
		case "$p" in
		"$report_dir"/*)
			refuse "commit ${sha:0:12} on $wip changes $p: a report under $report_dir/ never lands on the default branch."
			;;
		"$LOCAL_TIER"/* | "$LOCAL_TIER")
			refuse "commit ${sha:0:12} on $wip carries $p from $LOCAL_TIER/, the local tier, which never travels: a receipt or a handover made on another machine is never read here."
			;;
		esac
	done <<<"$(paths "$sha")"
done <<<"$work"

# The tooling: what preflight runs, and as whom. preflight on the collected
# tree runs that tree's code as the person, so a change to the build file, the
# scripts, the module files, or a top-level hidden directory (the hooks in
# .githooks/, the workflows in .github/, a host's settings directory; every one
# but .abcd/, the records) would let the cloud session decide which gates run,
# or mint a receipt without running any. The hidden directories are matched by
# shape, so no host is named here (C12). Refused unless the person says
# --allow-tooling-changes, so by default the gates that run are the default
# branch's own.
is_tooling() {
	case "$1" in
	Makefile | GNUmakefile | makefile | go.mod | go.sum | scripts/*) return 0 ;;
	.abcd/*) return 1 ;;
	.*/*) return 0 ;;
	esac
	return 1
}
tooling=""
while IFS= read -r sha; do
	while IFS= read -r p; do
		if [ -n "$p" ] && is_tooling "$p"; then
			tooling="${tooling:+$tooling
}$p"
		fi
	done <<<"$(paths "$sha")"
done <<<"$work"
if [ -n "$tooling" ] && [ -z "$allow_tooling" ]; then
	tooling="$(printf '%s\n' "$tooling" | sort -u)"
	echo "collect: the session's commits change the tooling collect would run as you:" >&2
	# shellcheck disable=SC2046 # one path per line, none with a space (git quotes those)
	git diff --stat=200 --no-renames "$fork" "$tip^" -- $(printf '%s\n' "$tooling") >&2 || true
	printf '%s\n' "$tooling" | sed 's/^/  changed: /' >&2
	refuse "the work changes the tooling (the build file, scripts/, go.mod, go.sum or a top-level hidden directory such as .githooks/ or .github/), and preflight on the collected tree would run that code as you. Read the changes above; collect them only by passing --allow-tooling-changes."
fi

# The attribution gate over the work alone, the report commit left out. Its
# findings are printed: they name commits, never a machine of the person's.
rc=0
bash "$root/scripts/check-attribution.sh" commits "$fork" "$tip^" >&2 || rc=$?
[ "$rc" -eq 0 ] ||
	refuse "the session's commits fail the attribution gate (scripts/check-attribution.sh commits, exit $rc; above). Collect never rewrites authorship: the commits must already name the person as author and carry Assisted-by:."

# --- 3. The worktree in the machine store ----------------------------------------

[ -n "${HOME:-}" ] && [ "$HOME" != / ] || refuse "HOME is not set, so the machine store cannot be found."
abcd_home="$HOME/.abcd.noindex"
root_sha="$(git rev-list -n 1 --max-parents=0 HEAD 2>/dev/null)" || root_sha=""
[[ "$root_sha" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]] ||
	refuse "this repository's root commit cannot be read, so its place in the machine store is unknown."
store="$abcd_home/worktrees/$root_sha"
# Every level of the store, not just the first: a symlink at any of them would
# put the worktree, and the preflight run in it, wherever the link points.
for level in "$abcd_home" "$abcd_home/worktrees" "$store"; do
	[ ! -L "$level" ] || refuse "$level is a symlink; the machine store is never read or written through one."
done
wt="$store/collect-$item"
[ ! -e "$wt" ] && [ ! -L "$wt" ] ||
	refuse "$wt already exists, kept by an earlier collect; inspect it, then retire it with git worktree remove and git branch -D $landing."
! git rev-parse --verify --quiet "refs/heads/$landing" >/dev/null ||
	refuse "the branch $landing already exists; retire it (and its worktree) before collecting again."

mkdir -p "$store"
chmod 700 "$abcd_home" "$abcd_home/worktrees" "$store" 2>/dev/null || true
git worktree add -q -b "$landing" "$wt" "$base" >&2 ||
	refuse "git could not make the worktree $wt."

# From here on the worktree is kept, whatever happens, for inspection.
failed() {
	echo "collect: FAILED: $1" >&2
	echo "collect: nothing was pushed. The worktree is kept for inspection at $wt, on $landing." >&2
	exit 1
}

# No -x: a "cherry picked from" line would follow the trailers and hide them.
# The hooks are the person's own checkout's, by absolute path: a relative
# core.hooksPath, or a global dispatcher that chains the repository's
# .githooks, resolves inside the worktree, where a collected commit may have
# put hooks of its own.
# shellcheck disable=SC2086 # the list is commit names, one per word
if ! out="$(git -C "$wt" -c core.hooksPath="$root/.githooks" cherry-pick --allow-empty $work 2>&1)"; then
	printf '%s\n' "$out" >&2
	git -C "$wt" cherry-pick --abort >/dev/null 2>&1 || true
	failed "the session's commits do not apply cleanly onto origin's $default."
fi
head="$(git -C "$wt" rev-parse HEAD)"

# --- 4. Preflight, waiting its turn ------------------------------------------------

log_dir="$wt/$LOCAL_TIER/logs"
mkdir -p "$log_dir"
log="$log_dir/collect-preflight.log"
echo "collect: running make preflight on ${head:0:12} in $wt (waiting while another preflight runs on this machine)." >&2
rc=0
go run ./cmd/abcd implement load --wait -- make -C "$wt" preflight 2>&1 | tee "$log" >&2 || rc=$?
if [ "$rc" -ne 0 ]; then
	echo "collect: the preflight's last lines (the whole run is in $log):" >&2
	tail -n 40 "$log" >&2
	failed "make preflight failed (exit $rc) on the collected commits."
fi

# --- 5. The offer ---------------------------------------------------------------------

receipt="$wt/$LOCAL_TIER/preflight-receipts/$head"
[ "$(git -C "$wt" rev-parse HEAD)" = "$head" ] ||
	failed "HEAD moved in the worktree during the preflight, so no single commit is what it read."
if [ ! -f "$receipt" ] || git -C "$wt" ls-files --error-unmatch -- "$LOCAL_TIER/preflight-receipts/$head" >/dev/null 2>&1; then
	failed "make preflight passed but minted no receipt for ${head:0:12} (see $log), so a push would be refused."
fi

printf 'collect: preflight passed on this computer. Nothing was pushed.\n'
printf '  branch:   %s (at %s, in %s)\n' "$landing" "${head:0:12}" "$wt"
printf '  receipt:  %s\n' "$receipt"
printf '  to land it, push it yourself and open a pull request:\n'
printf '    git -C %q push origin %s\n' "$wt" "$landing"
