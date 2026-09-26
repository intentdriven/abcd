#!/usr/bin/env bash
# Proves check-issue-resolution.sh can actually FAIL.
#
# A gate is an enforcement claim, and an enforcement claim is a fact about the
# code or it is nothing. The failure mode this guards against is the one
# iss-2608230847432286 records as its sharpest shape: a real gate, defended by a
# test that cannot fail, so the hole it leaves is invisible precisely because
# something green is watching it. Every rule here is asserted in BOTH directions
# — a violating fixture must be refused, and a clean one must pass — because a
# check that only ever sees clean input proves nothing about refusal.
#
# Fixtures are built in a scratch repository, never against this one: the rules
# are about ledger moves and commit reachability, and both are cheap to stage
# and impossible to stage honestly in a tree someone is working in.
#
# Usage: check-issue-resolution-cases.sh
# Exit 0 all cases behaved, 1 a case did not.
set -euo pipefail

# Hermetic git (iss-28, iss-313): every scratch-repo command below is `git -C
# "$d" …`, but an inherited absolute GIT_DIR overrides -C and redirects these
# fixture commits onto the ambient repository — which then reports all-green while
# its real history is rewritten. An inherited GIT_CONFIG_GLOBAL / core.hooksPath
# also fires the developer's global hooks inside the scratch repos and breaks the
# run for a reason that has nothing to do with the ledger. Neutralise the ambient
# git environment before the first git call, matching check-attribution-cases.sh
# and gitutil.IsolatedEnv. (commit.gpgsign is set per-repo below; this closes the
# rest.)
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
	GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0

GATE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-issue-resolution.sh"
[ -x "$GATE" ] || {
	echo "cases: gate not executable: $GATE" >&2
	exit 2
}

failures=0
tmproot="$(mktemp -d)"
trap 'rm -rf "$tmproot"' EXIT

ISS_DIR=".abcd/work/issues"

# expect <want: pass|fail> <repo> <label> -- <gate args...>
#
# The repo is a parameter rather than something the caller cds into, because a
# caller wrapping this in a subshell would discard the failure counter and leave
# THIS script unable to fail — the very defect it exists to catch. The cd lives
# inside the command substitution, where it cannot outlive the capture and the
# exit status still reaches the parent shell.
expect() {
	local want="$1" repo="$2" label="$3"
	shift 4
	local out rc=0
	out="$(cd "$repo" && bash "$GATE" "$@" 2>&1)" || rc=$?
	case "$want" in
	pass)
		if [ "$rc" -ne 0 ]; then
			printf 'cases: FAIL %s — expected clean, got exit %d:\n%s\n' "$label" "$rc" "$out" >&2
			failures=$((failures + 1))
		else
			printf 'cases: ok   %s (clean, as expected)\n' "$label"
		fi
		;;
	fail)
		if [ "$rc" -eq 0 ]; then
			printf 'cases: FAIL %s — the gate PASSED a violating fixture. This rule cannot fail.\n%s\n' "$label" "$out" >&2
			failures=$((failures + 1))
		else
			printf 'cases: ok   %s (refused, as expected)\n' "$label"
		fi
		;;
	esac
}

# expect_refusal_naming <repo> <label> <pattern> -- <gate args...>
#
# The message half of a refusal. On a branch 235 commits behind main RS001's
# exit code was right and its text was wrong (iss-2609012023256534): 84
# violations, each telling the reader to resolve a record the base already held
# as terminal, and not one line naming the rebase that was the only remedy. A
# gate that refuses for the right reason and diagnoses the wrong one sends the
# reader to fix the wrong thing, so these cases assert the diagnosis as well as
# the refusal: exit 1, AND the output carries the pattern.
expect_refusal_naming() {
	local repo="$1" label="$2" pattern="$3"
	shift 4
	local out rc=0
	out="$(cd "$repo" && bash "$GATE" "$@" 2>&1)" || rc=$?
	if [ "$rc" -ne 1 ]; then
		printf 'cases: FAIL %s — expected a refusal (exit 1), got exit %d:\n%s\n' "$label" "$rc" "$out" >&2
		failures=$((failures + 1))
	elif ! printf '%s\n' "$out" | grep -qE -- "$pattern"; then
		printf 'cases: FAIL %s — refused, but the message does not carry the diagnosis (want /%s/):\n%s\n' "$label" "$pattern" "$out" >&2
		failures=$((failures + 1))
	else
		printf 'cases: ok   %s (refused, naming the diagnosis)\n' "$label"
	fi
}

# expect_refusal_not_naming <repo> <label> <pattern> -- <gate args...>
#
# The mirror: a refusal whose message must NOT carry the pattern. A remedy that
# is right for one shape and wrong for its neighbour is the drift this pins —
# a rebase cures a stale branch and cures nothing for a trailer that names an
# issue resolved before the branch ever diverged.
expect_refusal_not_naming() {
	local repo="$1" label="$2" pattern="$3"
	shift 4
	local out rc=0
	out="$(cd "$repo" && bash "$GATE" "$@" 2>&1)" || rc=$?
	if [ "$rc" -ne 1 ]; then
		printf 'cases: FAIL %s — expected a refusal (exit 1), got exit %d:\n%s\n' "$label" "$rc" "$out" >&2
		failures=$((failures + 1))
	elif printf '%s\n' "$out" | grep -qE -- "$pattern"; then
		printf 'cases: FAIL %s — refused, but the message names a remedy that does not apply (/%s/):\n%s\n' "$label" "$pattern" "$out" >&2
		failures=$((failures + 1))
	else
		printf 'cases: ok   %s (refused, without the misleading remedy)\n' "$label"
	fi
}

# newrepo makes a scratch repo with one baseline commit and an open issue.
newrepo() {
	local d="$tmproot/$1"
	mkdir -p "$d/$ISS_DIR/open" "$d/$ISS_DIR/resolved" "$d/$ISS_DIR/wontfix"
	git -C "$d" init -q -b main
	git -C "$d" config user.name t
	git -C "$d" config user.email t@example.invalid
	git -C "$d" config commit.gpgsign false
	cat >"$d/$ISS_DIR/open/iss-999-a-fixture.md" <<'EOF'
---
schema_version: 1
id: "iss-999"
---
A fixture issue.
EOF
	# .gitkeep so the empty destination directories survive the baseline commit.
	touch "$d/$ISS_DIR/resolved/.gitkeep" "$d/$ISS_DIR/wontfix/.gitkeep"
	git -C "$d" add -A
	git -C "$d" commit -qm "baseline"
	# The change under test goes on a branch: with everything on main, main..HEAD
	# is empty and the gate correctly reports "nothing to check" — which a naive
	# fixture reads as a pass.
	git -C "$d" checkout -q -b work
	echo "$d"
}

resolve_record() {
	local d="$1" sha="${2:-}"
	git -C "$d" mv "$ISS_DIR/open/iss-999-a-fixture.md" "$ISS_DIR/resolved/iss-999-a-fixture.md"
	if [ -n "$sha" ]; then
		python3 - "$d/$ISS_DIR/resolved/iss-999-a-fixture.md" "$sha" <<'PY'
import sys
p, sha = sys.argv[1], sys.argv[2]
s = open(p).read()
s = s.replace('id: "iss-999"\n', 'id: "iss-999"\nresolved_by:\n  commit: "%s"\n' % sha)
open(p, "w").write(s)
PY
	fi
}

# --- RS001: a declared resolution must move the record -----------------------

d="$(newrepo rs001-bad)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
# The gate reads the CWD's repo, so run it there.
expect fail "$d" "RS001 trailer, record left in open/" -- commits main HEAD

d="$(newrepo rs001-good)"
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
expect pass "$d" "RS001 trailer with the ledger move" -- commits main HEAD

# A resolution with no trailer is legitimate — a stale issue resolved on its own
# merits has no fixing commit to name — so the gate must NOT demand the reverse.
d="$(newrepo rs001-noclaim)"
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve a stale issue"
expect pass "$d" "ledger move with no trailer is allowed" -- commits main HEAD

# A bare `git rm` of the open record with a Resolves: trailer is NOT a
# resolution: the record leaves open/ but ENTERS neither resolved/ nor wontfix/,
# so it vanishes from the ledger and its changelog line is lost. RS001 keys on
# the id ENTERING a terminal folder, so a delete-only "resolution" is refused.
d="$(newrepo rs001-delete-only)"
git -C "$d" rm -q "$ISS_DIR/open/iss-999-a-fixture.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
expect fail "$d" "RS001 trailer with a bare delete of the record" -- commits main HEAD

# A rewrite-move that stays inside open/ is not a resolution either, however
# git reports it: rewritten heavily enough it is a D plus an A, and the D half
# must not read as a terminal landing.
d="$(newrepo rs001-renamed-within-open)"
git -C "$d" mv "$ISS_DIR/open/iss-999-a-fixture.md" "$ISS_DIR/open/iss-999-a-fixture-renamed.md"
for i in 1 2 3 4 5 6 7 8 9 10; do
	echo "Rewritten line $i so git reports the move as a delete plus an add." >>"$d/$ISS_DIR/open/iss-999-a-fixture-renamed.md"
done
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
expect fail "$d" "RS001 trailer, record renamed within open/" -- commits main HEAD

# The mirror of the bare delete, and the load-bearing case: a record CAPTURED and
# resolved in the same change lands as a plain add into resolved/ — a two-dot
# base..head diff shows no departure from open/, because the record was never in
# open/ at the base. It MUST pass: this is how the bug-hunt loop resolves its own
# findings. RS001 keys on entering a terminal folder, not on leaving open/, so it
# holds. (A blanket intersection of leaves-open and enters-closed would fail this
# and break every such PR at the push gate.)
d="$(newrepo rs001-fresh-capture)"
cat >"$d/$ISS_DIR/resolved/iss-777-born-resolved.md" <<'EOF'
---
schema_version: 1
id: "iss-777"
---
A finding captured and resolved in one change.
EOF
git -C "$d" add -A
git -C "$d" commit -qm "fix: address a fresh finding

Resolves: iss-777"
expect pass "$d" "RS001 trailer with a same-change capture-and-resolve" -- commits main HEAD

# --- RS001 on a stale branch: the diagnosis, not just the refusal -------------
#
# The shape that produced 84 misdirected violations (iss-2609012023256534): the
# branch resolved its issue honestly, the work was squash- or rebase-merged so
# the BASE now holds the moved record too, and the branch was never rebased. The
# two-dot diff of the two trees shows the record entering nothing, because both
# trees already hold it in resolved/, so the trailer goes unsatisfied within the
# range. The refusal stands — a rebase makes the range honest, and the merged
# commits vanish from it — but the message must say what the script can prove:
# the record is already terminal at the base, which base-side commit put it
# there, how far behind the head is, and that a rebase is the remedy.
d="$(newrepo rs001-stale-branch)"
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
# The squash merge of that same work, landing on main after the branch diverged.
git -C "$d" checkout -q main
git -C "$d" mv "$ISS_DIR/open/iss-999-a-fixture.md" "$ISS_DIR/resolved/iss-999-a-fixture.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something (squash of work)"
git -C "$d" checkout -q work
expect_refusal_naming "$d" "RS001 on a stale branch names the base-side resolution and a rebase" \
	"already sits in $ISS_DIR/resolved/ at main .*squash of work.*1 commit\\(s\\) behind main.*[Rr]ebase onto main" -- commits main HEAD

# The neighbour a rebase does NOT cure: the record was terminal at the merge
# base, before the branch diverged, and a branch commit still claims to resolve
# it. The trailer is simply wrong, and the remedy is to drop it. The base holds
# an unrelated commit the head lacks, so a behind-count alone cannot tell the
# two apart — only the base-side history of the record can.
d="$(newrepo rs001-terminal-before-divergence)"
git -C "$d" checkout -q main
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve a stale issue"
git -C "$d" checkout -q -B work main
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something else

Resolves: iss-999"
git -C "$d" checkout -q main
echo "unrelated" >"$d/unrelated.txt"
git -C "$d" add -A
git -C "$d" commit -qm "chore: unrelated base-side commit"
git -C "$d" checkout -q work
expect_refusal_naming "$d" "RS001 on a record terminal before divergence says to drop the trailer" \
	"already sat in $ISS_DIR/resolved/ before this branch diverged from main.*[Dd]rop the trailer" -- commits main HEAD
expect_refusal_not_naming "$d" "RS001 on a record terminal before divergence does not prescribe a rebase" \
	"[Rr]ebase" -- commits main HEAD

# A trailer naming a record the head tree does not hold at all, while the base
# does: the branch predates the record (a cherry-pick from main onto a stale
# branch produces exactly this). The rebase brings the record; the message must
# say the record is missing HERE and present THERE, not "resolve it in this
# change", which cannot be done for a record the tree lacks.
d="$(newrepo rs001-record-absent-here)"
git -C "$d" checkout -q main
cat >"$d/$ISS_DIR/resolved/iss-555-born-on-main.md" <<'EOF'
---
schema_version: 1
id: "iss-555"
---
A record captured and resolved on main after the branch diverged.
EOF
git -C "$d" add -A
git -C "$d" commit -qm "fix: on main

Resolves: iss-555"
git -C "$d" checkout -q work
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: replayed onto the stale branch

Resolves: iss-555"
expect_refusal_naming "$d" "RS001 on a record absent from the head names the base's copy and a rebase" \
	"iss-555 has no record at HEAD, while main holds it in $ISS_DIR/resolved/.*[Rr]ebase onto main" -- commits main HEAD

# A trailer naming an id no ref holds: a typo, or an uncommitted capture. No
# rebase helps; the message must say the record exists nowhere.
d="$(newrepo rs001-record-absent-everywhere)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-404"
expect_refusal_naming "$d" "RS001 on an id with no record anywhere says so" \
	"iss-404 has no record at HEAD or at main" -- commits main HEAD
expect_refusal_not_naming "$d" "RS001 on an id with no record anywhere does not prescribe a rebase" \
	"[Rr]ebase" -- commits main HEAD

# The stale-branch diagnosis on a record whose slug carries a non-ASCII byte.
# `git ls-tree --name-only` C-quotes such a path ("…/iss-998-\303\251.md",
# quotes included), so a locator reading it unquoted takes the opening quote as
# the status folder and the diagnosis silently falls back to the generic text
# (iss-2609012047552618). Same topology as the stale-branch case above; the
# only difference is the slug.
d="$(newrepo rs001-stale-branch-nonascii)"
cat >"$d/$ISS_DIR/open/iss-998-é.md" <<'EOF'
---
schema_version: 1
id: "iss-998"
---
A fixture issue with an accented slug.
EOF
git -C "$d" add -A
git -C "$d" commit -qm "chore: capture with an accented slug"
git -C "$d" checkout -q main
git -C "$d" merge -q --ff-only work
git -C "$d" checkout -q work
git -C "$d" mv "$ISS_DIR/open/iss-998-é.md" "$ISS_DIR/resolved/iss-998-é.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-998"
git -C "$d" checkout -q main
git -C "$d" mv "$ISS_DIR/open/iss-998-é.md" "$ISS_DIR/resolved/iss-998-é.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something (squash of work)"
git -C "$d" checkout -q work
expect_refusal_naming "$d" "RS001 on a stale branch diagnoses a record with a non-ASCII slug" \
	"iss-998 already sits in $ISS_DIR/resolved/ at main .*squash of work.*[Rr]ebase onto main" -- commits main HEAD

# --- RS002: a stamp added here must name a reachable commit ------------------

d="$(newrepo rs002-bad)"
resolve_record "$d" "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve with a sha that does not exist"
expect fail "$d" "RS002 stamp naming a nonexistent commit" -- commits main HEAD

d="$(newrepo rs002-good)"
base="$(git -C "$d" rev-parse HEAD)"
resolve_record "$d" "$base"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve citing a real ancestor"
expect pass "$d" "RS002 stamp naming a reachable commit" -- commits main HEAD

# The unreachable-but-real case: a sha that exists in the repo on an unrelated
# branch the head cannot see. This is the squash/rebase shape, staged honestly.
d="$(newrepo rs002-unreachable)"
git -C "$d" checkout -q -b sidebranch main
echo side >"$d/side.txt"
git -C "$d" add -A
git -C "$d" commit -qm "side commit"
side="$(git -C "$d" rev-parse HEAD)"
git -C "$d" checkout -q work
resolve_record "$d" "$side"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve citing an unreachable commit"
expect fail "$d" "RS002 stamp naming a real but unreachable commit" -- commits main HEAD

# --- RS006: a resolution names only tests that exist -------------------------

# resolve_with_note moves the fixture into resolved/ with a resolution naming
# the tests given, and stages a test file defining the ones listed in $3.
resolve_with_note() {
	local d="$1" note="$2" defined="$3"
	resolve_record "$d"
	python3 - "$d/$ISS_DIR/resolved/iss-999-a-fixture.md" "$note" <<'PY'
import sys
p, note = sys.argv[1], sys.argv[2]
s = open(p).read()
s = s.replace('id: "iss-999"\n', 'id: "iss-999"\nresolution: "%s"\n' % note)
open(p, "w").write(s)
PY
	if [ -n "$defined" ]; then
		mkdir -p "$d/pkg"
		{
			echo "package pkg"
			for t in $defined; do printf '\nfunc %s(t *testing.T) {}\n' "$t"; done
		} >"$d/pkg/x_test.go"
	fi
	git -C "$d" add -A
	git -C "$d" commit -qm "chore: resolve"
}

d="$(newrepo rs006-bad)"
resolve_with_note "$d" "fixed; TestRealGuard and TestInventedGuard pin it" "TestRealGuard"
expect_refusal_naming "$d" "RS006 resolution naming a test no file defines" \
	"RS006 iss-999's resolution names TestInventedGuard" -- commits main HEAD

d="$(newrepo rs006-good)"
resolve_with_note "$d" "fixed; TestRealGuard pins it" "TestRealGuard"
expect pass "$d" "RS006 resolution naming a test that exists" -- commits main HEAD

d="$(newrepo rs006-none)"
resolve_with_note "$d" "fixed by rewording the message" ""
expect pass "$d" "RS006 resolution naming no test" -- commits main HEAD

# --- RS003: the ledger's existing stamps stay reachable ----------------------

d="$(newrepo rs003-good)"
base="$(git -C "$d" rev-parse HEAD)"
resolve_record "$d" "$base"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve"
expect pass "$d" "RS003 all stamps reachable" -- ledger HEAD

d="$(newrepo rs003-bad)"
resolve_record "$d" "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve"
expect fail "$d" "RS003 stamp naming a nonexistent commit" -- ledger HEAD

# --- boundary/robustness regressions ----------------------------------------

# RS002 reads the frontmatter only: a `commit:` line in a record's PROSE BODY is
# narrative, not a stamp, and must not be reachability-checked. The frontmatter
# here carries a real reachable stamp; the body documents the stamp shape with a
# nonexistent sha. Before the boundary fix the raw-diff scan flagged the body sha.
d="$(newrepo rs002-body-example)"
base="$(git -C "$d" rev-parse HEAD)"
resolve_record "$d" "$base"
cat >>"$d/$ISS_DIR/resolved/iss-999-a-fixture.md" <<'EOF'

Example of the stamp this rule checks:

    resolved_by:
      commit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
EOF
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve and document the stamp shape"
expect pass "$d" "RS002 ignores a commit: example in the prose body" -- commits main HEAD

# A non-iss file entering resolved/ or wontfix/ (a README, a policy note) is not a
# record, and it must not abort the gate. The id extractors skip a non-iss
# basename; before the fix that skip exited non-zero and, when it was the last
# diff entry under errexit, killed the whole run silently with RS001/RS002 never
# reached. zzz- sorts last, so it is that final entry.
d="$(newrepo rs001-nonrecord-last)"
resolve_record "$d"
echo "policy notes" >"$d/$ISS_DIR/wontfix/zzz-policy.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: something

Resolves: iss-999"
expect pass "$d" "a non-iss file last in the diff does not abort the gate" -- commits main HEAD

# The gate resolves its paths from the repository root, so it scans the same
# records from any working directory. Before the cd fix, a run from a subdirectory
# matched nothing and reported a clean pass over an unreachable-stamp ledger.
d="$(newrepo rs003-subdir)"
resolve_record "$d" "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve"
mkdir -p "$d/sub/deep"
expect fail "$d/sub/deep" "RS003 still scans when run from a subdirectory" -- ledger HEAD

# A git probe that fails must exit 2 as an environment fault, never read as an
# empty ledger. Before the rc-check fix, `cd "$(git rev-parse --show-toplevel)"`
# collapsed to a successful `cd ""` and the ls-tree || true turned the failure
# into "no ledger records — nothing to check", exit 0 — a vacuous pass from the
# one gate that notices rewritten resolution stamps. Two shapes: no repository
# at all, and a repository git refuses to read (dubious ownership, the form a
# container/devcontainer uid split produces on a real checkout).
d="$tmproot/not-a-repo"
mkdir -p "$d"
out="$(cd "$d" && bash "$GATE" ledger HEAD 2>&1)" && rc=0 || rc=$?
if [ "$rc" -ne 2 ]; then
	printf 'cases: FAIL a non-repository cwd must exit 2, got exit %d:\n%s\n' "$rc" "$out" >&2
	failures=$((failures + 1))
else
	printf 'cases: ok   a failing git probe is an environment fault, not an empty ledger\n'
fi

d="$(newrepo git-refused)"
resolve_record "$d" "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
git -C "$d" add -A
git -C "$d" commit -qm "chore: resolve"
out="$(cd "$d" && GIT_TEST_ASSUME_DIFFERENT_OWNER=1 bash "$GATE" ledger HEAD 2>&1)" && rc=0 || rc=$?
if [ "$rc" -ne 2 ]; then
	printf 'cases: FAIL a git-refused repository must exit 2, got exit %d:\n%s\n' "$rc" "$out" >&2
	failures=$((failures + 1))
else
	printf 'cases: ok   a git-refused repository (dubious ownership) is an environment fault\n'
fi

# The legitimate empty case survives the hardening: git succeeds, zero records —
# a loud "nothing to check" pass, not a fault.
d="$tmproot/empty-ledger"
mkdir -p "$d"
git -C "$d" init -q -b main
git -C "$d" config user.name t
git -C "$d" config user.email t@example.invalid
git -C "$d" config commit.gpgsign false
echo x >"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "baseline"
expect pass "$d" "an actually-empty ledger still passes loudly" -- ledger HEAD

# --- the sibling record families are outside the gate's scope ----------------
#
# The ledger root now holds readings/<run-id>/ and dispositions/<item-id>/, whose
# files are NOT issue records. RS002 and RS003 read the frontmatter of every .md
# under the pathspec and treat an indented `commit:` key as a resolution stamp, so
# an unscoped gate would reachability-check a value in one of these and refuse a
# clean tree. The gate scopes to the status directories; these two fixtures are
# what prove it, in the only direction that can regress — a record the gate must
# ignore.
#
# The frontmatter below deliberately carries that indented `commit:` shape. The
# record schemas do not mint one today, and that is exactly why the fixture spells
# it out: the gate reads BYTES under a pathspec, not schemas, so what protects
# these families is the scope and nothing else. A fixture that relied on today's
# field list would pass unscoped and prove nothing.

d="$(newrepo reading-record-ignored)"
mkdir -p "$d/$ISS_DIR/readings/rdg-1"
cat >"$d/$ISS_DIR/readings/rdg-1/rdi-2.md" <<'EOF'
---
schema_version: 1
id: "rdi-2"
run: "rdg-1"
manifest: "sha256:beef"
position: "detection"
regime: "registrative"
pattern: "a stated constraint"
tension: "the two sides disagree"
constraint_in_play: "the stated invariant"
why_a_tension: "one of them must give"
cited_by:
  commit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
---
EOF
git -C "$d" add -A
git -C "$d" commit -qm "chore: ingest a reading run"
expect pass "$d" "a reading record is outside the gate's scope" -- ledger HEAD
expect pass "$d" "a reading record is outside the commits scan too" -- commits main HEAD

d="$(newrepo disposition-ignored)"
mkdir -p "$d/$ISS_DIR/dispositions/rdi-2"
cat >"$d/$ISS_DIR/dispositions/rdi-2/dsp-3.md" <<'EOF'
---
schema_version: 1
id: "dsp-3"
item: "rdi-2"
state: "accepted"
disposition_grounds: "worth acting on"
cited_by:
  commit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
---
EOF
git -C "$d" add -A
git -C "$d" commit -qm "chore: answer a reading item"
expect pass "$d" "a disposition is outside the gate's scope" -- ledger HEAD
expect pass "$d" "a disposition is outside the commits scan too" -- commits main HEAD

d="$(newrepo admission-ignored)"
mkdir -p "$d/$ISS_DIR/admissions/rdg-1" "$d/$ISS_DIR/surprises"
cat >"$d/$ISS_DIR/admissions/rdg-1/adm-4.md" <<'EOF'
---
schema_version: 1
id: "adm-4"
run: "rdg-1"
proposal: "rdi-2"
grounds: "the configuration it admits is one the frame does not hold"
cited_by:
  commit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
---
EOF
cat >"$d/$ISS_DIR/surprises/srp-5.md" <<'EOF'
---
schema_version: 1
id: "srp-5"
occasioned_by: "rdi-2"
cited_by:
  commit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
---
EOF
git -C "$d" add -A
git -C "$d" commit -qm "chore: admit a proposal and record a surprise"
expect pass "$d" "the step-2 records are outside the gate's scope" -- ledger HEAD
expect pass "$d" "the step-2 records are outside the commits scan too" -- commits main HEAD

# --- RS004: a named record id must declare its relation ----------------------
#
# The rule (iss-2609100507421759): a commit message or a pull-request title/body
# that NAMES an iss-N must say what the change is to it. `Resolves: iss-N` says
# it fixes it, and RS001 above then requires the record to move in the same
# change; `Refs: iss-N` says touched-but-not-fixed and requires nothing of the
# ledger. A bare mention — the shape that let four fixed issues sit open in a
# managed repository's ledger, with the only evidence buried in commit prose —
# is refused here, before the merge.

d="$(newrepo rs004-bare-mention)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "fix: the parser hole behind iss-999"
expect_refusal_naming "$d" "RS004 bare mention in a commit subject" \
	"RS004.*iss-999" -- commits main HEAD

# The load-bearing case the rule turns on: `Refs:` is INFORMATIONAL. It declares
# the relation (so RS004 is satisfied) and must NOT drag RS001's move
# requirement along with it — a commit that merely touches an issue's ground
# leaves the record exactly where it was.
d="$(newrepo rs004-refs-no-move)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "refactor: tidy the parser around the iss-999 ground

Refs: iss-999"
expect pass "$d" "RS004 Refs: declares the relation and demands no ledger move" -- commits main HEAD

# The other declaration, already RS001's: it satisfies RS004 too, so the two
# rules cannot double-refuse one honest commit.
d="$(newrepo rs004-resolves-declares)"
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "fix: close the iss-999 hole

Resolves: iss-999"
expect pass "$d" "RS004 Resolves: is itself a declaration" -- commits main HEAD

# ONE spelling, deliberately. `Ref:`, `References:`, `See:` are not the trailer;
# admitting near-misses reopens the omission the rule closes, so a near-miss
# reads as what it is — an undeclared mention.
d="$(newrepo rs004-near-miss-trailer)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "refactor: tidy the parser

Ref: iss-999"
expect_refusal_naming "$d" "RS004 a near-miss trailer is not a declaration" \
	"RS004.*iss-999" -- commits main HEAD

# A merge commit's message is composed by the forge (`Merge pull request #N from
# …`) and can carry the branch's own text; the commits scan skips merges for the
# same reason RS001 does, so a mention inherited from a merged branch — already
# judged on that branch — is not re-refused here.
d="$(newrepo rs004-merge-exempt)"
git -C "$d" checkout -q -b side
echo "side" >>"$d/SIDE.md"
git -C "$d" add -A
git -C "$d" commit -qm "chore: side work

Refs: iss-999"
git -C "$d" checkout -q work
echo "work" >>"$d/WORK.md"
git -C "$d" add -A
git -C "$d" commit -qm "chore: work"
git -C "$d" merge -q --no-ff side -m "Merge branch 'side' — carries the iss-999 note"
expect pass "$d" "RS004 skips a merge commit's forge-composed message" -- commits main HEAD

# A declaration may name more than one id on one line: `Refs: iss-1, iss-2` is
# the conventional trailer shape, and refusing it made the author write the
# trailer twice or — the failure the rule exists to close — drop the second id.
# The vocabulary stays closed at two spellings; only the id LIST widens.
d="$(newrepo rs004-refs-list)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "refactor: tidy the ground around iss-999 and iss-998

Refs: iss-999, iss-998"
expect pass "$d" "RS004 a comma-separated Refs: list declares every id on the line" -- commits main HEAD

# The list form must not become an escape from RS001: every id a `Resolves:`
# line names is a declared resolution, so every one of them must move. RS001
# reads the whole line for that reason, not just its first id — a rule that read
# one id would let a comma carry the others past the move requirement.
d="$(newrepo rs004-resolves-list)"
resolve_record "$d"
git -C "$d" add -A
git -C "$d" commit -qm "fix: close the iss-999 hole

Resolves: iss-999, iss-998"
expect_refusal_naming "$d" "RS001 reads every id in a Resolves: list" \
	"RS001.*declares 'Resolves: iss-998'" -- commits main HEAD

# --- RS004 on the pull-request form ------------------------------------------
#
# The same rule over the artefact a commit message cannot reach: the title and
# body a human types into the forge. The declaration lives in the BODY (a title
# has no room for a trailer), so the two are judged against one declaration set.
#
# A PR body is also the one artefact a later run may no longer see. A SQUASH
# merge taken outside the merge queue composes its commit from the pull-request
# TITLE plus the branch's commit bodies and drops the PR body entirely — so a
# mention that lives only in the title, declared only in the body, passes here
# and then fails RS004 on the post-merge commit, which carries the title's
# mention and none of the body's declaration. This repository's queue merges
# rather than squashes, so it bites only on a squash taken outside it; the
# remedy is to put the declaration where the mention is.

d="$(newrepo rs004-pr-title-bare)"
printf '%s' "fix: the parser hole behind iss-999" >"$d/title.txt"
printf '%s\n' "Tidies the parser." >"$d/body.md"
expect_refusal_naming "$d" "RS004 bare mention in a pull-request title" \
	"RS004.*title.*iss-999|RS004.*iss-999" -- pr title.txt body.md

d="$(newrepo rs004-pr-body-bare)"
printf '%s' "fix: the parser hole" >"$d/title.txt"
printf '%s\n' "Tidies the parser; the ground is the one iss-999 describes." >"$d/body.md"
expect_refusal_naming "$d" "RS004 bare mention in a pull-request body" \
	"RS004.*iss-999" -- pr title.txt body.md

d="$(newrepo rs004-pr-declared)"
printf '%s' "fix: the parser hole behind iss-999" >"$d/title.txt"
printf '%s\n' "Tidies the parser.

Refs: iss-999" >"$d/body.md"
expect pass "$d" "RS004 a title mention declared in the body" -- pr title.txt body.md

# A non-UTF-8 byte anywhere on a line must not hide the mention on it. Under a
# UTF-8 locale grep drops a line holding an invalid byte ENTIRELY (verified on
# BSD grep) and tr refuses it outright, so a title or body carrying one — a
# Latin-1 accent from an editor that never converted — got a silent pass from
# the whole rule. The gate pins LC_ALL=C for exactly that, so this case runs it
# under a UTF-8 locale on purpose. It is staged on the pull-request form rather
# than on a commit message because git transcodes a message it judges
# non-conforming, which would settle the fixture per platform instead of per
# rule.
d="$(newrepo rs004-latin1-byte)"
printf 'fix: the caf\xe9 parser hole behind iss-999' >"$d/title.txt"
printf '%s\n' "Tidies the parser." >"$d/body.md"
utf8_locale="$(locale -a 2>/dev/null | grep -iE '^(en_US|C)\.(utf-?8)$' | head -1 || true)"
saved_lc_all="${LC_ALL-}"
if [ -n "$utf8_locale" ]; then export LC_ALL="$utf8_locale"; fi
expect_refusal_naming "$d" "RS004 a mention on a line carrying a non-UTF-8 byte" \
	"RS004.*iss-999" -- pr title.txt body.md
if [ -n "$saved_lc_all" ]; then export LC_ALL="$saved_lc_all"; else unset LC_ALL; fi

d="$(newrepo rs004-pr-clean)"
printf '%s' "chore: tidy the parser" >"$d/title.txt"
printf '%s\n' "No record is named here." >"$d/body.md"
expect pass "$d" "RS004 a pull-request form naming no record" -- pr title.txt body.md

# --- RS005: a declared delivery must ship the intent --------------------------
#
# The intent-store twin of RS001 (itd-2609111003026787). A change that delivers
# an intent says so with `Delivers: itd-N`, and the named intent must ENTER
# .abcd/development/intents/shipped/ in the same range — which `abcd spec close`
# does as its close-hook, on the close after which no open spec names it. No
# trailer, no assertion: a planned intent nobody claims to have built is the
# ordinary state of the backlog, and the standing population of planned intents
# with open specs is deliberately out of this rule's reach.

INT_DIR=".abcd/development/intents"
SPC_DIR=".abcd/development/specs"

# newrepo_intents is newrepo plus a small intent store on main: itd-7 planned
# with spc-7 open, itd-8 planned with no spec (spec_id: null), itd-6 shipped
# with spc-6 closed, itd-9 a draft, itd-5 superseded.
newrepo_intents() {
	local d
	d="$(newrepo "$1")"
	git -C "$d" checkout -q main
	mkdir -p "$d/$INT_DIR/drafts" "$d/$INT_DIR/planned" "$d/$INT_DIR/shipped" \
		"$d/$INT_DIR/disciplines" "$d/$INT_DIR/superseded" "$d/$SPC_DIR/open" "$d/$SPC_DIR/closed"
	intent_fixture "$d" planned 7 spc-7
	intent_fixture "$d" planned 8 null
	intent_fixture "$d" shipped 6 spc-6
	intent_fixture "$d" drafts 9 null
	intent_fixture "$d" superseded 5 null
	spec_fixture "$d" open 7 itd-7
	spec_fixture "$d" closed 6 itd-6
	touch "$d/$INT_DIR/disciplines/.gitkeep"
	git -C "$d" add -A
	git -C "$d" commit -qm "baseline intents"
	git -C "$d" checkout -q -B work main
	echo "$d"
}

intent_fixture() {
	local d="$1" bucket="$2" n="$3" spec="$4"
	printf -- '---\nid: itd-%s\nslug: fixture-%s\nspec_id: %s\n---\n\n# Fixture intent %s\n' \
		"$n" "$n" "$spec" "$n" >"$d/$INT_DIR/$bucket/itd-$n-fixture-$n.md"
}

spec_fixture() {
	local d="$1" status="$2" n="$3" intent="$4"
	printf -- '---\nid: spc-%s\nslug: fixture-%s\nintent: %s\n---\n\n# Fixture spec %s\n' \
		"$n" "$n" "$intent" "$n" >"$d/$SPC_DIR/$status/spc-$n-fixture-$n.md"
}

# ship_intent stages what `abcd spec close spc-N` does to the tree: the spec
# moves open/ -> closed/, and the intent planned/ -> shipped/.
ship_intent() {
	local d="$1" n="$2" spec="${3:-$2}"
	git -C "$d" mv "$SPC_DIR/open/spc-$spec-fixture-$spec.md" "$SPC_DIR/closed/spc-$spec-fixture-$spec.md"
	git -C "$d" mv "$INT_DIR/planned/itd-$n-fixture-$n.md" "$INT_DIR/shipped/itd-$n-fixture-$n.md"
}

# Criterion 1: a declared delivery whose intent stays planned is refused, and the
# refusal names the command that closes its spec.
d="$(newrepo_intents rs005-left-planned)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
expect_refusal_naming "$d" "RS005 trailer, intent left in planned/ with its spec open" \
	"RS005 commit [0-9a-f]{12} declares 'Delivers: itd-7', but itd-7 does not enter $INT_DIR/shipped/ in main\\.\\.HEAD.*abcd spec close spc-7" -- commits main HEAD

d="$(newrepo_intents rs005-good)"
echo "touched" >>"$d/README.md"
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
expect pass "$d" "RS005 trailer with the spec closed and the intent shipped" -- commits main HEAD

# The close may land in a later commit of the same change than the trailer: the
# rule reads the range, as RS001 does.
d="$(newrepo_intents rs005-close-in-later-commit)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "chore: close spc-7"
expect pass "$d" "RS005 trailer with the close in a later commit of the range" -- commits main HEAD

# Criterion 2: several declared, every unshipped one is named in the one run —
# whether they share a comma-separated line or stand on lines of their own.
d="$(newrepo_intents rs005-several)"
intent_fixture "$d" planned 4 spc-4
spec_fixture "$d" open 4 itd-4
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build three things

Delivers: itd-7, itd-8
Delivers: itd-4"
expect_refusal_naming "$d" "RS005 names the first of several unshipped intents" \
	"declares 'Delivers: itd-7'" -- commits main HEAD
expect_refusal_naming "$d" "RS005 names the second of several unshipped intents" \
	"declares 'Delivers: itd-8'" -- commits main HEAD
expect_refusal_naming "$d" "RS005 names an unshipped intent on its own trailer line" \
	"declares 'Delivers: itd-4'" -- commits main HEAD
expect_refusal_naming "$d" "RS005 counts every unshipped intent as its own violation" \
	"FAILED — 3 violation\\(s\\)" -- commits main HEAD

# Criterion 3 and 6: no trailer, no refusal — even with planned intents whose
# specs are open all around, and even when the change touches one of them.
d="$(newrepo_intents rs005-no-claim)"
echo "touched" >>"$d/README.md"
echo "An edit to the planned record." >>"$d/$INT_DIR/planned/itd-7-fixture-7.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: part of the thing"
expect pass "$d" "RS005 no trailer, planned intents with open specs are not refused" -- commits main HEAD

# Criterion 4: a bad id says which.
d="$(newrepo_intents rs005-absent-everywhere)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-404"
expect_refusal_naming "$d" "RS005 on an id with no record anywhere says so" \
	"itd-404 has no record at HEAD or at main" -- commits main HEAD
expect_refusal_not_naming "$d" "RS005 on an id with no record anywhere does not prescribe a rebase" \
	"[Rr]ebase" -- commits main HEAD

d="$(newrepo_intents rs005-already-shipped)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing again

Delivers: itd-6"
expect_refusal_naming "$d" "RS005 on an intent already shipped before the branch says to drop the trailer" \
	"itd-6 already sat in $INT_DIR/shipped/ before this branch diverged from main.*[Dd]rop the trailer" -- commits main HEAD
expect_refusal_not_naming "$d" "RS005 on an intent already shipped before the branch does not prescribe a rebase" \
	"[Rr]ebase" -- commits main HEAD

# The stale-branch shape: the branch shipped the intent honestly, the work was
# squash-merged so main holds the shipped record too, and the branch was never
# rebased. Both trees hold it in shipped/, so it enters nothing in the range.
d="$(newrepo_intents rs005-stale-branch)"
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
git -C "$d" checkout -q main
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing (squash of work)"
git -C "$d" checkout -q work
expect_refusal_naming "$d" "RS005 on a stale branch names the base-side ship and a rebase" \
	"itd-7 already sits in $INT_DIR/shipped/ at main .*squash of work.*[Rr]ebase onto main" -- commits main HEAD

d="$(newrepo_intents rs005-absent-here)"
git -C "$d" checkout -q main
intent_fixture "$d" planned 3 spc-3
spec_fixture "$d" open 3 itd-3
git -C "$d" add -A
git -C "$d" commit -qm "docs: plan itd-3 on main"
git -C "$d" checkout -q work
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-3"
expect_refusal_naming "$d" "RS005 on an intent absent from the head names the base's copy and a rebase" \
	"itd-3 has no record at HEAD, while main holds it in $INT_DIR/planned/.*[Rr]ebase onto main" -- commits main HEAD

# The spec table's last row: an intent planned with no spec has none to close,
# and a refusal naming `abcd spec close` would name a command with no argument.
d="$(newrepo_intents rs005-no-spec)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-8"
expect_refusal_naming "$d" "RS005 on a planned intent with no spec says it has none to close" \
	"itd-8 .*no spec to close" -- commits main HEAD
expect_refusal_not_naming "$d" "RS005 on a planned intent with no spec names no spec close" \
	"abcd spec close" -- commits main HEAD

# A draft and a superseded record are not delivered by closing anything.
d="$(newrepo_intents rs005-draft)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-9"
expect_refusal_naming "$d" "RS005 on a draft says it is unplanned" \
	"itd-9 .*$INT_DIR/drafts/.*not been planned" -- commits main HEAD

d="$(newrepo_intents rs005-superseded)"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-5"
expect_refusal_naming "$d" "RS005 on a superseded intent says so" \
	"itd-5 .*$INT_DIR/superseded/" -- commits main HEAD

# 1:n specs (adr-2609151513118583): an intent ships on the close after which no
# open spec names it. Closing one spec while a remainder stays open leaves the
# intent planned, so the refusal names the spec still open — found by the spec's
# own `intent:` back-link, not by the intent's scalar spec_id.
d="$(newrepo_intents rs005-remainder-open)"
spec_fixture "$d" open 70 itd-7
git -C "$d" add -A
git -C "$d" commit -qm "docs: a remainder spec for itd-7"
git -C "$d" mv "$SPC_DIR/open/spc-7-fixture-7.md" "$SPC_DIR/closed/spc-7-fixture-7.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: part of the thing

Delivers: itd-7"
expect_refusal_naming "$d" "RS005 names the remainder spec still open" \
	"itd-7 .*abcd spec close spc-70" -- commits main HEAD
expect_refusal_not_naming "$d" "RS005 does not name a spec already closed" \
	"spec close spc-7[^0-9]" -- commits main HEAD

# Ids are compared canonically (recordid.SameID): a zero-padded trailer or
# back-link names the same record as the unpadded filename.
d="$(newrepo_intents rs005-padded)"
echo "touched" >>"$d/README.md"
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-007"
expect pass "$d" "RS005 a zero-padded trailer names the same intent" -- commits main HEAD

d="$(newrepo_intents rs005-padded-backlink)"
spec_fixture "$d" open 71 itd-007
git -C "$d" add -A
git -C "$d" commit -qm "docs: a remainder spec with a padded back-link"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
expect_refusal_naming "$d" "RS005 finds an open spec through a zero-padded back-link" \
	"abcd spec close spc-71" -- commits main HEAD

# The fail-closed half of the canonical comparison (recordid.SameID): a
# back-link that is not an intent id at all — a bare number, or `null` — names
# no intent, so it is never offered as the spec to close.
d="$(newrepo_intents rs005-bare-backlink)"
spec_fixture "$d" open 72 7
spec_fixture "$d" open 73 null
git -C "$d" add -A
git -C "$d" commit -qm "docs: specs whose back-links name no intent"
echo "touched" >>"$d/README.md"
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-7"
expect_refusal_not_naming "$d" "RS005 does not follow a back-link that is not an intent id" \
	"spc-7[23]" -- commits main HEAD

# A `Delivers:` line the rule cannot read is refused rather than passed over: the
# spec id, or the trailer in the wrong case, would otherwise be a declaration the
# author believes armed and the gate never sees. What makes a line such an
# attempt is an id-shaped token in its value; a line without one is prose.
for spelling in "Delivers: spc-7" "delivers: itd-7" "Delivers: itd-7 and itd-8"; do
	d="$(newrepo_intents "rs005-malformed-$(printf '%s' "$spelling" | tr -c 'a-z0-9' '-')")"
	echo "touched" >>"$d/README.md"
	ship_intent "$d" 7
	git -C "$d" add -A
	git -C "$d" commit -qm "feat: build the thing

$spelling"
	expect_refusal_naming "$d" "RS005 refuses the unreadable trailer '$spelling'" \
		"RS005 commit [0-9a-f]{12} carries a delivery line RS005 cannot read.*Delivers: itd-N" -- commits main HEAD
done

# Criterion 3: a change that declares no delivery is refused nothing — and a body
# line that merely begins with the word is not a declaration. The second message
# is main's own 182474f5, whose wrapped prose puts `deliver:` at a line start;
# RS005 refused it until a near-miss had to carry an id-shaped token.
prose_bodies=(
	"feat: build the thing

Delivers: the thing"
	"fix: name the update verb from version --check and the schema-too-new refusals

itd-130 shipped \`abcd update\` and promised two things its surfaces did not
deliver: that \`version --check\` would follow \"update available\" with the
next line to type, and that the eight schema-too-new refusals would name a
verb instead of the verbless \"upgrade abcd\"."
)
for i in "${!prose_bodies[@]}"; do
	d="$(newrepo_intents "rs005-prose-$i")"
	echo "touched" >>"$d/README.md"
	git -C "$d" add -A
	git -C "$d" commit -qm "${prose_bodies[$i]}"
	expect pass "$d" "RS005 passes a prose line that starts with the word (body $i)" -- commits main HEAD
done

# An empty spec store's open/: the lookup for open specs naming the intent finds
# no file at all, and must answer "none" rather than end the run. Closing spc-7
# leaves open/ with nothing in it, and the declared itd-8 is planned with no spec.
d="$(newrepo_intents rs005-open-empty)"
echo "touched" >>"$d/README.md"
ship_intent "$d" 7
git -C "$d" add -A
git -C "$d" commit -qm "feat: build the thing

Delivers: itd-8"
expect_refusal_naming "$d" "RS005 with an empty open/ still diagnoses the planned intent" \
	"itd-8 .*no spec to close" -- commits main HEAD

# Criterion 5: the intent rule's refusal has the issue rule's shape and exit
# code — compared here, not judged by a reviewer. Both fixtures are the ordinary
# case (a trailer whose record stays where it was); each refusal is normalised by
# ONE pattern that fixes the shape, and the two must agree byte for byte after it,
# exit code included.
shape_of() {
	local repo="$1" out rc=0
	out="$(cd "$repo" && bash "$GATE" commits main HEAD 2>&1 >/dev/null)" || rc=$?
	printf 'exit=%d\n' "$rc"
	printf '%s\n' "$out" | sed -E \
		"s/^check-issue-resolution: RS[0-9]{3} commit [0-9a-f]{12} declares '[A-Z][a-z]+: [a-z]{3}-[0-9]+', but [a-z]{3}-[0-9]+ does not enter [^ ]+( or [^ ]+)? in main\\.\\.HEAD[.,] .* or drop the trailer\\.\$/check-issue-resolution: <RULE> commit <SHA> declares '<TRAILER>: <ID>', but <ID> does not enter <DEST> in <RANGE>. <REMEDY> or drop the trailer./"
}
d_iss="$(newrepo rs005-shape-issue)"
echo "touched" >>"$d_iss/README.md"
git -C "$d_iss" add -A
git -C "$d_iss" commit -qm "fix: something

Resolves: iss-999"
d_itd="$(newrepo_intents rs005-shape-intent)"
echo "touched" >>"$d_itd/README.md"
git -C "$d_itd" add -A
git -C "$d_itd" commit -qm "feat: something

Delivers: itd-7"
shape_iss="$(shape_of "$d_iss")"
shape_itd="$(shape_of "$d_itd")"
if [ "$shape_iss" != "$shape_itd" ]; then
	printf 'cases: FAIL RS005 refusal shape differs from RS001'"'"'s:\n--- RS001\n%s\n--- RS005\n%s\n' "$shape_iss" "$shape_itd" >&2
	failures=$((failures + 1))
elif ! printf '%s\n' "$shape_itd" | grep -q '<RULE> commit <SHA>' || ! printf '%s\n' "$shape_itd" | grep -qx 'exit=1'; then
	printf 'cases: FAIL RS005 refusal shape — the normaliser matched neither refusal, so the comparison proves nothing:\n%s\n' "$shape_itd" >&2
	failures=$((failures + 1))
else
	printf 'cases: ok   RS005 refusal has RS001'"'"'s message shape and exit code\n'
fi

if [ "$failures" -gt 0 ]; then
	printf 'cases: FAILED — %d case(s) did not behave\n' "$failures" >&2
	exit 1
fi
echo "cases: OK — every rule refused its violating fixture and passed its clean one"
