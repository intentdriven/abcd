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
#      rewrites authorship. Refuse, too, an added path whose case-folded form
#      is another path's in the tree the work lands in, the default branch's
#      paths less those the collected commits delete, with the copy's (on a
#      file system that folds case, SCRIPTS/gate.sh overwrites
#      scripts/gate.sh), whatever the flags say; a case-only rename deletes
#      the path it folds onto, so it passes. And refuse when the work
#      changes the tooling, the build and gate machinery: the build file,
#      scripts/, go.mod, go.sum, go.work, go.work.sum, vendor/, the gates'
#      settings and baselines at the top of .abcd/ (every .abcd/*.json, and
#      .abcd/config/), the files under .abcd/work/ a gate reads (the reviews
#      charter in reviews/, the ruleset mirror in rulesets/, and every
#      .abcd/work/*.json), and every top-level hidden directory but .abcd/
#      (the hooks, the workflows, a host's settings), printing the diffstat of
#      those paths, unless --allow-tooling-changes is passed. Paths are read
#      NUL-separated, so git never quotes one, and matched case-folded, so
#      neither a quoted name nor a capital steps round a pattern. The other
#      records under .abcd/development/ and .abcd/work/ (the issue records,
#      CONTEXT.md, the decision records in decisions/, intake.md) are no
#      tooling;
#   3. make a worktree in the machine store,
#      ~/.abcd.noindex/worktrees/<root-sha>/collect-<item-id>/
#      (adr-2610031751065746; <root-sha> is the full name of the repository's
#      root commit; no level of it may be a symlink), on a new branch
#      collect/<item-id> from origin's default branch, and cherry-pick the
#      work onto it, the report commits left out, with the hooks of the
#      person's own checkout (core.hooksPath by absolute path), never hooks a
#      collected commit put in the worktree;
#   4. print, on standard output, what is about to run and the full diffstat
#      of the collected commits against origin's default branch; then run
#      `make preflight` there, through
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
# What runs, and as whom: collect runs the collected copy's code on the
# person's computer, as the person. Preflight builds and runs that copy's Go
# code, its tests and its Go-written gates included (go test, go run
# ./cmd/abcd, the evals), with the person's credentials in reach. That is the
# cost decision 9 accepted, and step 4 says so before preflight starts, with
# the diffstat of what will run. The tooling refusal does not make the gates
# the default branch's: it only keeps the build and gate machinery (the build
# file, scripts/, the hooks, CI and the gates' settings) as the default branch
# has it, so the cloud session cannot choose which gates run or mint a receipt
# without running them. There is no confirmation prompt: collect is run on
# purpose, and what it is about to run is printed first.
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

tmpd="$(mktemp -d "${TMPDIR:-/tmp}/abcd-collect.XXXXXX")"
trap '/bin/rm -rf -- "$tmpd"' EXIT

# fold prints a path with its ASCII letters in lower case, the case-folding
# check-wip-source.sh applies, so a capital never steps round a pattern.
fold() {
	printf '%s' "$1" | LC_ALL=C tr 'A-Z' 'a-z'
}

# paths <commit> sets P to what one commit changes against its parent, one
# path per line. git writes them NUL-separated (-z), so it quotes none: a
# quoted name ("scripts/\303\251.sh") would match no pattern below. A path
# holding a line break is refused, since it could not be told from two.
P=""
paths() {
	local p
	P=""
	git diff-tree -z --no-commit-id --name-only -r --no-renames "$1" >"$tmpd/paths" ||
		refuse "git could not list what commit ${1:0:12} changes."
	while IFS= read -r -d '' p; do
		case "$p" in
		*$'\n'*) refuse "commit ${1:0:12} on $wip changes a path holding a line break; collect lands no such path." ;;
		esac
		P="${P:+$P
}$p"
	done <"$tmpd/paths"
}

parents="$(git rev-list --parents -n 1 "$tip" | wc -w | tr -d ' ')"
paths "$tip"
[ "$parents" -eq 2 ] && [ "$P" = "$report" ] ||
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
	paths "$sha"
	[ "$P" = "$report" ] && continue
	work="${work:+$work
}$sha"
done
[ -n "$work" ] ||
	refuse "$wip carries the report and no work: there is nothing to collect. Read the report with scripts/cloud/list.sh."

# The tooling: the build and gate machinery, which decides how the collected
# copy is built and which gates run on it. A change to it would let the cloud
# session choose the gates, or mint a receipt without running any: the build
# file, the scripts, the module and workspace files, a vendored dependency,
# the gates' settings and baselines at the top of .abcd/ (record-lint.json,
# docs-lint.json, site.json, the citation and site baselines, rules.json,
# config.json and the rest of .abcd/*.json, and .abcd/config/), the files
# under .abcd/work/ a gate reads (the reviews charter and its receipts in
# reviews/, read by lint-reviews and the release gate; the ruleset mirror in
# rulesets/, read by the required-check tests and the build loop; and every
# .abcd/work/*.json, such as brief-review-flags.json, read by record-lint),
# and a top-level hidden directory (the hooks in .githooks/, the workflows in
# .github/, a host's settings directory; every one but .abcd/, whose other
# records under development/ and work/ are no tooling). The hidden
# directories are matched by shape, so no host is named here (C12). The path
# is folded first.
is_tooling() {
	case "$1" in
	makefile | gnumakefile | go.mod | go.sum | go.work | go.work.sum | scripts/* | vendor/*) return 0 ;;
	.abcd/config/* | .abcd/work/reviews/* | .abcd/work/rulesets/*) return 0 ;;
	.abcd/work/*/*) return 1 ;;
	.abcd/work/*.json) return 0 ;;
	.abcd/*/*) return 1 ;;
	.abcd/*.json) return 0 ;;
	.abcd/*) return 1 ;;
	.*/*) return 0 ;;
	esac
	return 1
}

report_f="$(fold "$report_dir")"
tier_f="$(fold "$LOCAL_TIER")"
tooling=""
: >"$tmpd/added"
: >"$tmpd/deleted"
while IFS= read -r sha; do
	paths "$sha"
	while IFS= read -r p; do
		[ -n "$p" ] || continue
		f="$(fold "$p")"
		case "$f" in
		"$report_f"/*)
			refuse "commit ${sha:0:12} on $wip changes $p: a report under $report_dir/ never lands on the default branch."
			;;
		"$tier_f"/* | "$tier_f")
			refuse "commit ${sha:0:12} on $wip carries $p from $LOCAL_TIER/, the local tier, which never travels: a receipt or a handover made on another machine is never read here."
			;;
		esac
		if is_tooling "$f"; then
			tooling="${tooling:+$tooling
}$p"
		fi
	done <<<"$P"
	git diff-tree -z --no-commit-id --name-only -r --no-renames --diff-filter=A "$sha" | tr '\0' '\n' >>"$tmpd/added" ||
		refuse "git could not list what commit ${sha:0:12} adds."
	git diff-tree -z --no-commit-id --name-only -r --no-renames --diff-filter=D "$sha" | tr '\0' '\n' >>"$tmpd/deleted" ||
		refuse "git could not list what commit ${sha:0:12} deletes."
done <<<"$work"

if [ -n "$tooling" ] && [ -z "$allow_tooling" ]; then
	tooling="$(printf '%s\n' "$tooling" | LC_ALL=C sort -u)"
	echo "collect: the session's commits change the build and gate machinery collect would run as you:" >&2
	specs=()
	while IFS= read -r p; do specs+=("$p"); done <<<"$tooling"
	git --literal-pathspecs diff --stat=200 --no-renames "$fork" "$tip^" -- "${specs[@]}" >&2 || true
	printf '%s\n' "$tooling" | LC_ALL=C tr -d '\001-\011\013-\037\177' | sed 's/^/  changed: /' >&2
	refuse "the work changes the tooling (the build file, scripts/, the module or workspace files, vendor/, the gates' settings at the top of .abcd/, the files under .abcd/work/ a gate reads, or a top-level hidden directory such as .githooks/ or .github/), which decides how the collected copy is built and gated as you. Read the changes above; collect them only by passing --allow-tooling-changes."
fi

# Case collisions: an added path whose case-folded form is another path's in
# the tree the work lands in would overwrite that path on a file system that
# folds case. That tree is the default branch's paths less those the collected
# commits delete, with the copy's tree (another added path among them): a
# case-only rename (seed.md to Seed.md, an add under --no-renames) deletes the
# path it folds onto, so it collides with nothing. Refused whatever the flags
# say: no flag makes an overwrite the person can read in a diffstat.
git ls-tree -r -z --name-only "$base" | tr '\0' '\n' >"$tmpd/base" ||
	refuse "git could not list the default branch's tree."
git ls-tree -r -z --name-only "$tip^" | tr '\0' '\n' >"$tmpd/copy" ||
	refuse "git could not list the copy's tree."
collide="$(LC_ALL=C awk '
	length($0) == 0 { next }
	FILENAME == ARGV[1] { added[$0] = 1; next }
	FILENAME == ARGV[2] { deleted[$0] = 1; next }
	FILENAME == ARGV[3] && ($0 in deleted) { next }
	!($0 in seen) { seen[$0] = 1; n[tolower($0)]++ }
	END { for (a in added) if (n[tolower(a)] > 1) print a }
' "$tmpd/added" "$tmpd/deleted" "$tmpd/base" "$tmpd/copy" | LC_ALL=C sort)"
if [ -n "$collide" ]; then
	printf '%s\n' "$collide" | LC_ALL=C tr -d '\001-\011\013-\037\177' | sed 's/^/  collides: /' >&2
	refuse "the work adds a path whose name differs only in case from another path's (above), and on a file system that folds case it would overwrite that path. Rename it in the session's branch."
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

# --- 4. What will run, then preflight, waiting its turn ----------------------------

# On standard output, before preflight starts: what runs, as whom, and the
# full diffstat of the collected commits against origin's default branch.
printf "collect: make preflight now runs the collected copy's code on this computer, as you: its Go code, its tests and its Go-written gates included (go test, go run ./cmd/abcd, the evals). That is the cost decision 9 accepted.\n"
if [ -n "$allow_tooling" ]; then
	printf "collect: --allow-tooling-changes was passed, so the build and gate machinery is the copy's too (the build file, scripts/, the hooks, CI and the gates' settings, as the session left them).\n"
else
	printf "collect: the tooling refusal only keeps the build and gate machinery (the build file, scripts/, the hooks, CI and the gates' settings) as origin's %s has it.\n" "$default"
fi
printf "collect: the collected commits change, against origin's %s:\n" "$default"
git -C "$wt" diff --stat=200 --no-renames "$base" "$head" ||
	failed "git could not show what the collected commits change."


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
