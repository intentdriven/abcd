#!/usr/bin/env bash
# Test corpus for check-wip-source.sh, the required check that keeps a cloud
# session's work-in-progress branch and its report off the default branch
# (spc-2610120941441282 step 7, itd-2610120738075547 C7, adr-2610120937435856,
# brief invariant 23).
#
# The cases run before the gate in CI, as every gate's cases do: a check whose
# refusals have never been watched is an enforcement claim with nothing behind
# it (iss-2608230847432286).
#
# Run: bash scripts/check-wip-source-cases.sh   (exit 0 all pass, 1 otherwise)
set -uo pipefail

# Hermetic git, as check-attribution-cases.sh sets it out: an inherited GIT_DIR
# would redirect the scratch repository's commits onto the ambient one, and an
# inherited global config could sign or skew them.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
	GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0

cd "$(git rev-parse --show-toplevel)"

SCRIPT="$PWD/scripts/check-wip-source.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

pass=0
fail=0

# The scratch repository: one base commit, and a branch per diff case. The
# identity is set in the environment and every call names git itself, so the
# shell guard reads each one for what it runs rather than a wrapper that could
# pass git any subcommand.
repo="$tmp/repo"
export GIT_AUTHOR_NAME="Case Person" GIT_AUTHOR_EMAIL=case@example.com
export GIT_COMMITTER_NAME="Case Person" GIT_COMMITTER_EMAIL=case@example.com
git init -q -b main "$repo"
printf 'seed\n' >"$repo/seed.md"
printf 'notes\n' >"$repo/notes.md"
mkdir -p "$repo/.abcd/work/cloud-reports"
printf 'an older report\n' >"$repo/.abcd/work/cloud-reports/old.md"
git -C "$repo" add -A
git -C "$repo" commit -q -m seed
base="$(git -C "$repo" rev-parse HEAD)"

# branch_with <name> <path> <content> commits one file on a new branch from
# the base and prints its tip.
branch_with() {
	git -C "$repo" checkout -q -b "$1" "$base"
	mkdir -p "$(dirname "$repo/$2")"
	printf '%s\n' "$3" >"$repo/$2"
	git -C "$repo" add -A
	git -C "$repo" commit -q -m "$1"
	git -C "$repo" rev-parse HEAD
}

clean="$(branch_with clean src/x.go 'package x')"
report="$(branch_with report .abcd/work/cloud-reports/x.md 'a report')"
nested="$(branch_with nested .abcd/work/cloud-reports/sub/x.md 'a report')"
edited="$(branch_with edited .abcd/work/cloud-reports/old.md 'an edit')"
git -C "$repo" checkout -q -b renamed "$base"
git -C "$repo" mv notes.md .abcd/work/cloud-reports/notes.md
git -C "$repo" commit -q -m "move into the report dir"
renamed="$(git -C "$repo" rev-parse HEAD)"
git -C "$repo" checkout -q -b removed "$base"
git -C "$repo" rm -q .abcd/work/cloud-reports/old.md
git -C "$repo" commit -q -m "remove a report"
removed="$(git -C "$repo" rev-parse HEAD)"

# case_is <want:accept|reject|fault> <label> <source-ref> <base> <head>
case_is() {
	local want="$1" label="$2" got rc=0
	(cd "$repo" && bash "$SCRIPT" "$3" "$4" "$5") >"$tmp/out" 2>&1 || rc=$?
	case "$rc" in
	0) got=accept ;;
	1) got=reject ;;
	*) got=fault ;;
	esac
	if [ "$got" = "$want" ]; then
		pass=$((pass + 1))
	else
		fail=$((fail + 1))
		echo "FAIL: $label — wanted $want, got $got (exit $rc)" >&2
		sed 's/^/  /' "$tmp/out" >&2
	fi
}

# --- The source branch: the segment abcd-wip anywhere is refused ---------------
case_is reject "the segment at the top" "abcd-wip/iss-0000000000000000" "$base" "$clean"
case_is reject "the segment one deep" "test/abcd-wip/iss-0000000000000000" "$base" "$clean"
case_is reject "the segment one deep, as a full ref" "refs/heads/test/abcd-wip/iss-0000000000000000" "$base" "$clean"
case_is reject "the segment two deep" "a/b/abcd-wip/iss-0000000000000000" "$base" "$clean"
case_is reject "the segment last" "test/abcd-wip" "$base" "$clean"
case_is reject "the segment in upper case" "Testhost/ABCD-WIP/x" "$base" "$clean"
case_is reject "the segment in mixed case" "test/Abcd-Wip/iss-0000000000000000" "$base" "$clean"
case_is reject "the segment in upper case, as a full ref" "refs/heads/X/ABCD-WIP/y" "$base" "$clean"
case_is accept "a longer word that starts like it" "test/abcd-wipe/iss-0000000000000000" "$base" "$clean"
case_is accept "a shorter word" "wip/iss-0000000000000000" "$base" "$clean"
case_is accept "the word inside a segment" "feat/no-abcd-wip-here" "$base" "$clean"
case_is accept "an ordinary branch" "feat/cloud-sessions" "$base" "$clean"

# --- The diff: a file added under the report directory is refused -------------
case_is reject "a report added" "feat/x" "$base" "$report"
case_is reject "a report added one level down" "feat/x" "$base" "$nested"
case_is reject "a file moved into the report directory" "feat/x" "$base" "$renamed"
case_is accept "a clean diff" "feat/x" "$base" "$clean"
case_is accept "an existing report edited" "feat/x" "$base" "$edited"
case_is accept "an existing report removed" "feat/x" "$base" "$removed"
case_is reject "both at once" "test/abcd-wip/iss-0000000000000000" "$base" "$report"

# --- Faults fail closed ---------------------------------------------------------
case_is fault "no source branch named" "" "$base" "$clean"
case_is fault "a base that resolves to nothing" "feat/x" "0123456789abcdef0123456789abcdef01234567" "$clean"
case_is fault "the wrong number of arguments" "feat/x" "$base" ""

# --- The report directory is the one the cloud scripts write --------------------
# The check names its directory itself rather than reading cloud.conf from the
# tree it judges, so a change cannot move the guard by editing the setting; this
# case is what keeps the two from drifting apart.
conf_dir="$(sed -n 's/^report_dir=//p' scripts/cloud/cloud.conf)"
guard_dir="$(sed -n "s/^REPORT_DIR='\(.*\)'$/\1/p" "$SCRIPT")"
if [ -n "$conf_dir" ] && [ "$conf_dir" = "$guard_dir" ]; then
	pass=$((pass + 1))
else
	fail=$((fail + 1))
	echo "FAIL: scripts/cloud/cloud.conf's report_dir is '$conf_dir', but the check guards '$guard_dir'" >&2
fi

echo "check-wip-source-cases: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
