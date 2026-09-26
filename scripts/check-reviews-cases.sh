#!/usr/bin/env bash
# Proves check-reviews.sh can actually FAIL, and fails for the right reasons.
#
# RD002 is an append-only claim over committed history, and the claim is a fact
# about the history or it is nothing. Its first implementation probed each
# working-tree file with a pathspec-scoped `git log --diff-filter=MR`, which can
# never report an R — rename detection needs both sides in view — and swallowed
# git's own errors, so a rename and a broken git both read as clean. That is the
# worst shape a gate can take: green while it sees nothing. Every rule here is
# asserted in BOTH directions — a violating fixture must be refused, and a clean
# one must pass — because a check that only ever sees clean input proves nothing
# about refusal.
#
# Fixtures are built in a scratch repository, never against this one: the rule is
# about committed history, and history is cheap to stage in a throwaway repo and
# impossible to stage honestly in a tree someone is working in.
#
# Usage: check-reviews-cases.sh   (or /bin/bash check-reviews-cases.sh for bash 3.2)
# Exit 0 all cases behaved, 1 a case did not.
set -euo pipefail

# Hermetic git (iss-28, iss-313): every scratch-repo command below is `git -C
# "$d" …`, but an inherited absolute GIT_DIR overrides -C and redirects these
# fixture commits onto the ambient repository — which then reports all-green while
# its real history is rewritten. An inherited GIT_CONFIG_GLOBAL / core.hooksPath
# also fires the developer's global hooks inside the scratch repos and breaks the
# run for a reason that has nothing to do with the reviews charter. Neutralise the
# ambient git environment before the first git call, matching
# check-issue-resolution-cases.sh and gitutil.IsolatedEnv. (commit.gpgsign is set
# per-repo below; this closes the rest.)
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
	GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0

GATE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-reviews.sh"
[ -x "$GATE" ] || {
	echo "cases: gate not executable: $GATE" >&2
	exit 2
}

failures=0
tmproot="$(mktemp -d)"
trap 'rm -rf "$tmproot"' EXIT

REV_DIR=".abcd/work/reviews"
SCOPE="$REV_DIR/2026-01-01-a-scope"
# Any full object name serves as the fixture pin: RD004 judges the key's shape,
# and the board, not the gate, asks git how far behind it is.
PIN="0123456789abcdef0123456789abcdef01234567"

# expect <want: pass|fail|fault> <repo> <label> [needle]
#
# fault is exit 2 specifically — the refusal polarity reserved for "git could not
# answer", which must never collapse into either a pass or an ordinary RD failure.
# The optional needle is asserted against the gate's combined output, so a pass
# that is meant to say it covered nothing has to say so.
#
# The repo is a parameter rather than something the caller cds into, because a
# caller wrapping this in a subshell would discard the failure counter and leave
# THIS script unable to fail — the very defect it exists to catch. The cd lives
# inside the command substitution, where it cannot outlive the capture and the
# exit status still reaches the parent shell.
expect() {
	local want="$1" repo="$2" label="$3" needle="${4:-}"
	local out rc=0
	# "$BASH", not a bare bash: the gate runs under the interpreter running these
	# cases, so `/bin/bash scripts/check-reviews-cases.sh` proves it on the bash
	# 3.2 macOS ships and a bare invocation on whatever bash PATH names.
	out="$(cd "$repo" && "$BASH" "$GATE" 2>&1)" || rc=$?
	case "$want" in
	pass)
		if [ "$rc" -ne 0 ]; then
			printf 'cases: FAIL %s — expected clean, got exit %d:\n%s\n' "$label" "$rc" "$out" >&2
			failures=$((failures + 1))
			return
		fi
		;;
	fail)
		if [ "$rc" -eq 0 ]; then
			printf 'cases: FAIL %s — the gate PASSED a violating fixture. This rule cannot fail.\n%s\n' "$label" "$out" >&2
			failures=$((failures + 1))
			return
		fi
		;;
	fault)
		if [ "$rc" -ne 2 ]; then
			printf 'cases: FAIL %s — expected the environment-fault refusal (exit 2), got exit %d:\n%s\n' "$label" "$rc" "$out" >&2
			failures=$((failures + 1))
			return
		fi
		;;
	esac
	if [ -n "$needle" ] && ! printf '%s' "$out" | grep -qF "$needle"; then
		printf 'cases: FAIL %s — output does not say %q:\n%s\n' "$label" "$needle" "$out" >&2
		failures=$((failures + 1))
		return
	fi
	printf 'cases: ok   %s (exit %d, as expected)\n' "$label" "$rc"
}

# newrepo makes a scratch repo whose baseline commit holds a charter-clean review
# directory: RD001's <date>-<scope>/00-summary.md shape and no absolute paths, so
# only the RD002 polarity under test can move the verdict.
newrepo() {
	local d="$tmproot/$1"
	mkdir -p "$d/$SCOPE"
	git -C "$d" init -q -b main
	git -C "$d" config user.name t
	git -C "$d" config user.email t@example.invalid
	git -C "$d" config commit.gpgsign false
	printf -- '---\nreview_of_commit: %s\n---\n# Review summary\n\nA fixture review.\n' "$PIN" >"$d/$SCOPE/00-summary.md"
	printf '# Findings\n\nA fixture finding.\n' >"$d/$SCOPE/01-findings.md"
	printf '# Reviews charter\n\nThe root README is mutable.\n' >"$d/$REV_DIR/README.md"
	git -C "$d" add -A
	git -C "$d" commit -qm "baseline: add a review"
	echo "$d"
}

# --- RD002: the three shapes of a post-creation change -----------------------

# An in-place edit is the shape the original per-file probe did catch; it stays
# caught, so the rewrite is not a trade.
d="$(newrepo edit)"
printf 'An amendment made after the fact.\n' >>"$d/$SCOPE/01-findings.md"
git -C "$d" add -A
git -C "$d" commit -qm "edit a review file"
expect fail "$d" "RD002 in-place edit of a review file"

# A pure rename: git reports it as an R, which a pathspec-scoped log never emits.
# This fixture passed the original gate — the defect this script exists to pin.
d="$(newrepo rename)"
git -C "$d" mv "$SCOPE/01-findings.md" "$SCOPE/02-findings.md"
git -C "$d" commit -qm "rename a review file"
expect fail "$d" "RD002 pure rename of a review file"

# A rename with the content fully rewritten: past the similarity threshold git
# reports a delete plus an add rather than an R, so the D half must carry the
# refusal on its own.
d="$(newrepo rename-rewrite)"
git -C "$d" mv "$SCOPE/01-findings.md" "$SCOPE/03-findings.md"
: >"$d/$SCOPE/03-findings.md"
for i in 1 2 3 4 5 6 7 8 9 10; do
	echo "Rewritten line $i so git reports the move as a delete plus an add." >>"$d/$SCOPE/03-findings.md"
done
git -C "$d" add -A
git -C "$d" commit -qm "rename and rewrite a review file"
expect fail "$d" "RD002 rename with a full content rewrite"

# A deletion leaves no working-tree path to probe from, so a tree-driven check
# cannot see it at all. The history scan can.
d="$(newrepo delete)"
git -C "$d" rm -q "$SCOPE/01-findings.md"
git -C "$d" commit -qm "delete a review file"
expect fail "$d" "RD002 deletion of a review file"

# --- the clean polarity ------------------------------------------------------

# Add-only is what the charter asks for, and it must pass — a gate that refuses
# the intended shape is as useless as one that passes every shape.
d="$(newrepo clean)"
printf '# More findings\n\nAppended as a new file.\n' >"$d/$SCOPE/02-findings.md"
git -C "$d" add -A
git -C "$d" commit -qm "add another review file"
expect pass "$d" "add-only corpus is clean"

# The charter itself is not a review: reviews/README.md sits at the root, outside
# the dated-directory population, and stays mutable. Over-refusal is a real
# failure mode of a broadened scan, so it is asserted too.
d="$(newrepo readme)"
printf 'An amendment to the charter.\n' >>"$d/$REV_DIR/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "amend the charter"
expect pass "$d" "the root README stays mutable"

# --- RD004: every review names the commit it read (itd-28) --------------------

# addreview <repo> <dir-name> <summary-text> commits one new review folder.
addreview() {
	mkdir -p "$1/$REV_DIR/$2"
	printf '%s' "$3" >"$1/$REV_DIR/$2/00-summary.md"
	git -C "$1" add -A
	git -C "$1" commit -qm "add review $2"
}

# A review filed without the pin is refused: its age can never be told.
d="$(newrepo rd004-missing)"
addreview "$d" 2026-09-26-no-pin $'# Review summary\n\nNo frontmatter at all.\n'
expect fail "$d" "RD004 a review summary without review_of_commit" "RD004"

# The key in the body is not the key: only the leading frontmatter block counts,
# the board's one reading of it (internal/core/reviews.Pin).
d="$(newrepo rd004-body)"
addreview "$d" 2026-09-26-in-body "# Review summary

review_of_commit: $PIN
"
expect fail "$d" "RD004 the key in the body, not the frontmatter" "RD004"

# An abbreviated sha names no commit for certain, and a quoted one is a string
# the board does not read as a pin; both are refused, so gate and board agree.
d="$(newrepo rd004-short)"
addreview "$d" 2026-09-26-short $'---\nreview_of_commit: 0123456\n---\n# S\n'
expect fail "$d" "RD004 an abbreviated sha" "RD004"
d="$(newrepo rd004-quoted)"
addreview "$d" 2026-09-26-quoted "---
review_of_commit: \"$PIN\"
---
# S
"
expect fail "$d" "RD004 a quoted sha" "RD004"

# The pinned shape passes, a review of a spec among them (<date>-<spc-N>-<slug>),
# with a trailing comment and CRLF line ends, which the board reads the same way.
d="$(newrepo rd004-pinned)"
addreview "$d" 2026-09-26-spc-2609211854150455-rp-reviews "---"$'\r\n'"review_of_commit: $PIN # the tree read"$'\r\n'"---"$'\r\n'"# S"$'\r\n'
expect pass "$d" "RD004 a pinned review of a spec"

# A folder from before the rule is named as legacy, not refused. The legacy set
# is closed: these three names and no others.
d="$(newrepo rd004-legacy)"
addreview "$d" 2026-07-06-plan-consistency $'# Legacy summary\n'
expect pass "$d" "RD004 a legacy folder is named, not refused" "RD004 legacy"

# A sha-keyed receipt directory is pinned by its own name and has no summary.
d="$(newrepo rd004-receipt)"
mkdir -p "$d/$REV_DIR/$PIN"
printf '{}\n' >"$d/$REV_DIR/$PIN/gate.json"
git -C "$d" add -A
git -C "$d" commit -qm "add a receipt"
expect pass "$d" "RD004 a receipt directory is exempt"

# --- RD001/RD004: what the board cannot read, the gate refuses ----------------
#
# The board reads a summary through a guarded read (internal/core/reviews.Read):
# a symlink is no summary, and neither is one past maxSummaryBytes (1 MiB); both
# show unpinned. `[ -f ]` follows a symlink and the gate's read has no cap, so
# both were gate-green and board-unpinned. So was a NUL byte in the frontmatter:
# bash `read` drops it, so the gate saw a clean pin line where reviews.Pin sees
# none. Each is refused now, and each boundary's clean side is asserted too.

# pinned <file> writes a pinned summary's frontmatter and a short body.
pinned() {
	printf -- '---\nreview_of_commit: %s\n---\n# S\n' "$PIN" >"$1"
}

# padto <file> <bytes> grows a file with body text to exactly <bytes>.
padto() {
	local have
	have=$(($(wc -c <"$1")))
	head -c "$(($2 - have))" /dev/zero | tr '\0' 'a' >>"$1"
}

# commitall <repo> commits whatever the fixture wrote.
commitall() {
	git -C "$1" add -A
	git -C "$1" commit -qm "add a review"
}

# A symlinked summary is a pointer, not committed content: the board reads it as
# no summary at all.
d="$(newrepo rd001-symlink)"
mkdir -p "$d/$REV_DIR/2026-09-26-linked"
pinned "$d/$REV_DIR/2026-09-26-linked/01-real.md"
ln -s 01-real.md "$d/$REV_DIR/2026-09-26-linked/00-summary.md"
commitall "$d"
expect fail "$d" "RD001 a symlinked 00-summary.md" "RD001"

# A symlinked review folder is not on the board at all (the board does not follow
# a symlinked entry), so the gate refuses it rather than vouching for it.
d="$(newrepo rd001-symlink-dir)"
mkdir -p "$d/elsewhere"
pinned "$d/elsewhere/00-summary.md"
ln -s ../../../elsewhere "$d/$REV_DIR/2026-09-26-linked-dir"
commitall "$d"
expect fail "$d" "RD001 a symlinked review folder" "RD001"

# Past the board's 1 MiB cap the summary shows unpinned, so the gate refuses it;
# at exactly the cap the board reads it, and so does the gate.
d="$(newrepo rd004-oversize)"
mkdir -p "$d/$REV_DIR/2026-09-26-oversize"
pinned "$d/$REV_DIR/2026-09-26-oversize/00-summary.md"
padto "$d/$REV_DIR/2026-09-26-oversize/00-summary.md" 1048577
commitall "$d"
expect fail "$d" "RD004 a summary over 1 MiB" "RD004"
d="$(newrepo rd004-at-cap)"
mkdir -p "$d/$REV_DIR/2026-09-26-at-cap"
pinned "$d/$REV_DIR/2026-09-26-at-cap/00-summary.md"
padto "$d/$REV_DIR/2026-09-26-at-cap/00-summary.md" 1048576
commitall "$d"
expect pass "$d" "RD004 a summary of exactly 1 MiB"

# A NUL byte anywhere in the frontmatter block — after the sha, inside the key,
# or on the closing fence — leaves reviews.Pin with no pin; bash `read` drops it
# and would have read a clean block. The fixtures are written with printf's own
# escapes, since a shell variable cannot carry a NUL.
d="$(newrepo rd004-nul-value)"
mkdir -p "$d/$REV_DIR/2026-09-26-nul-value"
printf -- '---\nreview_of_commit: %s\0\n---\n# S\n' "$PIN" >"$d/$REV_DIR/2026-09-26-nul-value/00-summary.md"
commitall "$d"
expect fail "$d" "RD004 a NUL byte after the sha" "RD004"
d="$(newrepo rd004-nul-key)"
mkdir -p "$d/$REV_DIR/2026-09-26-nul-key"
printf -- '---\nreview_of\0_commit: %s\n---\n# S\n' "$PIN" >"$d/$REV_DIR/2026-09-26-nul-key/00-summary.md"
commitall "$d"
expect fail "$d" "RD004 a NUL byte inside the key" "RD004"
d="$(newrepo rd004-nul-fence)"
mkdir -p "$d/$REV_DIR/2026-09-26-nul-fence"
printf -- '---\nreview_of_commit: %s\n---\0\n# S\n' "$PIN" >"$d/$REV_DIR/2026-09-26-nul-fence/00-summary.md"
commitall "$d"
expect fail "$d" "RD004 a NUL byte on the closing fence" "RD004"
# The pin is read from the block alone, so a NUL in the body after it moves
# nothing: the board reads the pin, and the gate passes it.
d="$(newrepo rd004-nul-body)"
mkdir -p "$d/$REV_DIR/2026-09-26-nul-body"
printf -- '---\nreview_of_commit: %s\n---\n# S\0\n' "$PIN" >"$d/$REV_DIR/2026-09-26-nul-body/00-summary.md"
commitall "$d"
expect pass "$d" "RD004 a NUL byte in the body, past the block"

# --- environment polarities --------------------------------------------------

# An unborn HEAD is legitimate — a freshly scaffolded repo — and is the one git
# condition that passes. It must say loudly that it covered nothing, so the pass
# cannot be read as a verdict.
d="$tmproot/unborn"
mkdir -p "$d/$SCOPE"
git -C "$d" init -q -b main
printf '# Review summary\n\nUncommitted.\n' >"$d/$SCOPE/00-summary.md"
expect pass "$d" "unborn HEAD covers nothing, loudly" "no committed history"

# Every other git failure is an environment fault, not a clean history. Running
# outside a repository is the cheapest one to stage.
d="$tmproot/not-a-repo"
mkdir -p "$d"
expect fault "$d" "a git failure refuses instead of reading clean"

if [ "$failures" -gt 0 ]; then
	printf 'cases: FAILED — %d case(s) did not behave\n' "$failures" >&2
	exit 1
fi
echo "cases: OK — every rule refused its violating fixture and passed its clean one"
