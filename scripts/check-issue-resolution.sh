#!/usr/bin/env bash
# Deterministic gate for abcd's issue-resolution convention (AGENTS.md, the
# issue ledger under .abcd/work/issues/).
#
# `abcd capture resolve` exists and works. Nothing anywhere could fail when it
# was not run, so an issue that had in fact been fixed stayed in open/ until a
# human remembered, and a forgotten one left no marker to find it by: 328
# issues in resolved/ against 78 carrying resolved_by when this was measured.
# The resolution convention has no denominator of its own, because a
# fixed-but-unresolved issue is indistinguishable from an open one. Prose is not
# the missing piece; a check that fails closed is (iss-2608241347321757).
#
# The mechanic that forced the forgetting is resolved_by.commit. Every other
# field in a resolution is knowable while the record is being edited; the fixing
# commit's sha is not, because the record and the fix are the same change. That
# defers stamping past merge, and a deferred step is the one that gets dropped.
# So the gate's job is to make the resolution land INSIDE the change that fixes
# the issue, where no later step exists to forget.
#
# Three checks:
#
#   RS001  A `Resolves: iss-N` trailer in the range must be accompanied by that
#          issue ENTERING resolved/ or wontfix/ in the same range — whether it
#          moved out of open/ or was captured and resolved in the same change. A
#          commit that says it resolves an issue and lands the record in no
#          terminal folder is the exact drift this gate exists to stop.
#          The refusal names the shape it can prove (iss-2609012023256534): a
#          record already terminal at the base — the stale-branch shape, where a
#          rebase is the remedy and "resolve it" is not — is told apart from a
#          record left open, one the head tree lacks, and an id with no record.
#
#   RS002  A resolved_by.commit sha ADDED in the range must name a commit that
#          exists and is reachable from the head being pushed. The --commit flag
#          is shape-checked only (^[0-9a-f]{7,64}$), so a stamp naming a commit
#          that never existed reads exactly like a good one.
#
#   RS004  A commit message, or a pull-request title/body, that NAMES an iss-N
#          must declare its relation to it: `Resolves: iss-N` (this change fixes
#          it — RS001 then requires the record to move in the same change) or
#          `Refs: iss-N` (touched, not fixed — informational, demanding nothing
#          of the ledger). A bare mention is refused before the merge.
#          This is iss-2609100507421759's half that a gate can hold. Four fixed
#          issues sat open in a managed repository's ledger because the only
#          evidence of their fix was commit prose the ledger never reads; a
#          mention that must declare itself turns that prose into a signal both
#          this gate and `abcd capture mentions` can read. `Refs:` is
#          deliberately NOT a resolution: it is the escape that keeps the rule
#          honest, so a commit that merely touches an issue's ground is not
#          pushed into claiming a fix it did not make.
#
#          SCOPE. RS004 judges COMMIT MESSAGES and PULL-REQUEST TITLES/BODIES,
#          and nothing else. An id written in the BODY OF A RECORD is a
#          different surface with its own rule — `prose_citation_resolves`, in
#          internal/core/lint — because a record citing another record is making
#          a reference, not claiming to have changed anything. The gate is also
#          only the forward-looking half: it runs before a merge and cannot
#          reach the history a repository already has, which is what the
#          read-only `abcd capture mentions` listing reads.
#
#   RS005  A `Delivers: itd-N` trailer in the range must be accompanied by that
#          intent ENTERING .abcd/development/intents/shipped/ in the same range
#          (itd-2609111003026787) — the intent-store twin of RS001. `abcd spec
#          close` ships an intent as its close-hook, on the close after which no
#          open spec names it, and nothing runs that close for anyone: 61 planned
#          intents sat with their specs open when this was measured, and an
#          intent whose work is on main with its spec open ships with no
#          changelog line while the cut exits 0 (iss-2609091642508005). Nothing
#          here infers that work is live; the author declares it and the rule
#          holds the declaration, so a change naming no delivery is refused
#          nothing and the standing backlog is out of reach by design. The
#          trailer means "this change FINISHES the intent", not "contributes to
#          it". The refusal names the cause it can prove, in RS001's shape: an
#          id with no record, a branch that predates the record, an intent
#          already shipped, a draft or superseded record, a planned intent with
#          no spec to close, and — the ordinary case — every spec still open
#          that names it, each with its `abcd spec close`. A `Delivers:` line
#          that names an id the rule cannot read (a spec id, the wrong case) is
#          refused too, since an author who wrote it believes it armed; a line
#          naming no id-shaped token is prose, and passes.
#
#   RS006  A record entering resolved/ or wontfix/ in the range whose resolution
#          names a Go test (TestX) must name one some _test.go file defines at
#          the head (iss-2609020716579024). Records already terminal are not
#          re-read: a test renamed after the fact does not falsify a note.
#
#   RS003  Every resolved_by.commit already in the ledger must still be
#          reachable. This is the drift detector, and it is not hypothetical:
#          the repository allows merge, squash AND rebase, the method is a
#          per-pull-request choice, and under squash or rebase a cited branch
#          sha is rewritten out of existence. All 76 stamps were reachable when
#          this landed; RS003 is what notices the day one is not.
#
# Usage:
#   check-issue-resolution.sh commits <base-ref> <head-ref>   # RS001 + RS002 + RS004 + RS005 + RS006
#   check-issue-resolution.sh ledger [<ref>]                  # RS003 (default HEAD)
#   check-issue-resolution.sh pr <title-file> <body-file>     # RS004 on the PR form
#
# Exit 0 clean, 1 a violation, 2 a usage/environment fault.
set -euo pipefail

# Every scan below is over BYTES, not characters, so the whole script runs in the
# C locale. Under a UTF-8 locale a line carrying an invalid byte — a Latin-1
# accent from an editor that never converted, in a commit message or a pull-request
# body — is dropped ENTIRELY by grep (verified on BSD grep) and rejected outright
# by tr, so the mention on that line passes unseen and the gate reports a clean
# pass on the artefact it could not read. Nothing here is language-aware: the ids
# are ASCII and the declaration keywords are ASCII, so there is nothing a locale
# could usefully decide. Set once, at the top, so a scan added later inherits it.
export LC_ALL=C

# Resolve every path from the repository root, like the sibling gate
# check-reviews.sh. ISSUES_DIR and the git pathspecs below are relative, and a
# git pathspec is matched against the current directory — so run from a
# subdirectory the diff and ls-tree match nothing and the gate reports a clean
# pass having scanned zero records. cd first, so cwd cannot disarm it.
#
# Every git probe below is fail-closed: `cd "$(...)"` collapses to a successful
# `cd ""` when the substitution fails under errexit, and a swallowed git error
# reads exactly like an empty ledger — a gate that cannot tell them apart is a
# false green (git refuses a repo entirely on e.g. dubious ownership, so the
# fault is reachable from make preflight, not just a broken cwd).
rc=0
toplevel="$(git rev-parse --show-toplevel 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
	echo "check-issue-resolution: not a readable git repository (git rev-parse --show-toplevel exit $rc) — refusing rather than reporting a vacuous pass:" >&2
	echo "$toplevel" >&2
	exit 2
fi
cd "$toplevel"

# Every git call below lists or addresses paths, and git C-quotes a path holding
# a non-ASCII byte by default ("…/iss-998-\303\251.md", quotes included). A
# locator reading that unquoted takes the opening quote as the status folder,
# a `git show ref:path` on it finds nothing, and the gate degrades silently for
# exactly the records whose slug carries an accent (iss-2609012047552618). One
# wrapper turns quoting off for the whole script rather than per call, so a
# call added later cannot reintroduce it.
git() {
	command git -c core.quotePath=false "$@"
}

ISSUES_DIR=".abcd/work/issues"
# STATUS_DIRS is the ledger's status-directory list. The shell cannot import Go,
# so this is the second and LAST spelling of internal/core/issueschema's
# StatusDirs, pinned to it by TestIssueResolutionGateScopesToStatusDirs.
#
# The scan is scoped to these three rather than to the ledger root because the
# root holds sibling record families — readings/<run-id>/,
# dispositions/<item-id>/, admissions/<run-id>/ and surprises/ — whose files are
# not issue records, and the list of them grows. RS001 already
# matched only the terminal folders; RS002 and RS003 did not, and would read a
# `commit:`-shaped line out of one of those records as a resolution stamp.
STATUS_DIRS=(open resolved wontfix)
STATUS_PATHSPECS=()
for status_dir in "${STATUS_DIRS[@]}"; do
	STATUS_PATHSPECS+=("$ISSUES_DIR/$status_dir")
done
# The `Resolves:` trailer RS001 judges. Its id half is a comma-separated LIST, in
# step with DECLARE_RE below: one line may resolve several records, and every id
# on it is a declared resolution RS001 holds to the same move requirement. The ids
# are taken back out of the line with a second scan rather than from a capture
# group, because ERE has no repeated-group capture.
TRAILER_RE='^Resolves:[[:space:]]+iss-[0-9]+([[:space:]]*,[[:space:]]*iss-[0-9]+)*[[:space:]]*$'

# RS004's two spellings, and the mention scanner they are checked against.
#
# DECLARE_RE is the whole declaration vocabulary: `Resolves:` and `Refs:`,
# nothing but the declaration on the line. The ids are a COMMA-SEPARATED LIST,
# because `Refs: iss-1, iss-2` is the conventional trailer shape and refusing it
# made an author write the trailer twice or — the failure the rule exists to
# close — drop the second id. The VOCABULARY stays closed at two spellings; only
# the id list widens. `Ref:`, `References:`, `See:` and `Related:` are near-misses, and
# admitting them would reopen the omission the rule closes — the same argument
# that makes `Assisted-by: None` the only accepted non-vendor value in
# check-attribution.sh. A near-miss therefore reads as what it is: an
# undeclared mention, refused with the spelling named in the remedy.
#
# It shares TRAILER_RE's id shape rather than restating it, so the two rules
# cannot drift apart on what an id looks like: the `Resolves:` half of
# DECLARE_RE must match everything TRAILER_RE matches, or a commit RS001 judges
# would be a bare mention to RS004.
DECLARE_RE='^(Resolves|Refs):[[:space:]]+iss-[0-9]+([[:space:]]*,[[:space:]]*iss-[0-9]+)*[[:space:]]*$'

# MENTION_RE finds an id ANYWHERE in an artefact — subject line, prose body,
# trailer — because a mention is a mention wherever a reader meets it.
#
# The leading guard is a hand-rolled word boundary. POSIX ERE has none, `\b` is a
# GNU extension BSD grep does not share, and this gate runs on macOS as well as
# on CI: without the guard, `xiss-999` inside a longer token reads as a mention
# and the gate refuses a commit that names no record at all. The guard character
# is captured and stripped by the second grep rather than matched with a
# look-behind, which ERE also lacks.
MENTION_RE='(^|[^A-Za-z0-9])iss-[0-9]+'

# RS005's two stores. The shell cannot import Go, so these are the second
# spelling of intent.IntentsRelDir, intent.Buckets and spec.SpecsRelDir, pinned
# to them by TestIssueResolutionGateReadsTheIntentAndSpecStores. The intent
# lookups are scoped to the buckets for the reason STATUS_DIRS is: the store's
# root holds a README and other non-record files.
INTENTS_DIR=".abcd/development/intents"
INTENT_BUCKETS=(drafts planned shipped disciplines superseded)
SPECS_DIR=".abcd/development/specs"
INTENT_PATHSPECS=()
for bucket in "${INTENT_BUCKETS[@]}"; do
	INTENT_PATHSPECS+=("$INTENTS_DIR/$bucket")
done

# The `Delivers:` trailer RS005 judges: the intent id, as a comma-separated list
# in step with TRAILER_RE. `itd-[0-9]+` covers the sequential ids and the minted
# timestamp ids alike. It takes the INTENT id, never the spec id: the intent is
# the thing delivered, and a trailer that could name either store would be
# ambiguous about which one to look in.
DELIVERS_RE='^Delivers:[[:space:]]+itd-[0-9]+([[:space:]]*,[[:space:]]*itd-[0-9]+)*[[:space:]]*$'
# A line that reads as an attempt at the trailer: the word, whatever its case,
# then a colon (DELIVERS_LOOSE_RE), AND a record-id-shaped token anywhere in the
# value after that colon (DELIVERS_ID_SHAPED_RE). A line both match and
# DELIVERS_RE does not is refused rather than passed over: the author believes
# the declaration armed, and a gate that silently skipped it would be the
# omission the rule exists to close. The id-shaped token is what separates an
# attempt from prose: a wrapped body line can begin `deliver: that …`, and a
# change that declares no delivery is refused nothing (the intent's criterion
# 3), so a line naming no id is never a declaration. (Bracket classes, because
# bash 3.2 has neither ${var,,} nor nocasematch-safe portability here.)
DELIVERS_LOOSE_RE='^[Dd][Ee][Ll][Ii][Vv][Ee][Rr][Ss]?[[:space:]]*:'
DELIVERS_ID_SHAPED_RE='[A-Za-z]+-[0-9]+'

violations=0

fail() {
	printf 'check-issue-resolution: %s\n' "$1" >&2
	violations=$((violations + 1))
}

usage() {
	echo "usage: check-issue-resolution.sh commits <base-ref> <head-ref> | ledger [<ref>] | pr <title-file> <body-file>" >&2
	exit 2
}

# ids_entering_closed prints every iss-N whose record ENTERS resolved/ or wontfix/
# across the range — the destination half of a resolution. A record moved from
# open/ shows as a rename (or, without rename detection, as an add into the
# terminal folder); a record captured and resolved in the same change shows as a
# plain add into the terminal folder. Both are honest resolutions and both are
# caught here. A record that only LEAVES open/ (a bare delete) enters nothing and
# is deliberately absent, so RS001 refuses a trailer that merely deletes.
ids_entering_closed() {
	local base="$1" head="$2"
	git diff --name-status --find-renames "$base".."$head" -- "${STATUS_PATHSPECS[@]}" |
		while IFS=$'\t' read -r status path dest; do
			case "$status" in
			R*)
				case "$dest" in
				"$ISSUES_DIR/resolved/"* | "$ISSUES_DIR/wontfix/"*) basename "$dest" | grep -oE '^iss-[0-9]+' || true ;;
				esac
				;;
			A)
				case "$path" in
				"$ISSUES_DIR/resolved/"* | "$ISSUES_DIR/wontfix/"*) basename "$path" | grep -oE '^iss-[0-9]+' || true ;;
				esac
				;;
			esac
		done
}

# record_path prints the ledger path of iss-N's record at ref — its status
# folder is the diagnosis RS001 needs — or nothing when the ref holds none. The
# id is matched as a whole basename prefix, so iss-99 never answers for iss-999.
record_path() {
	local ref="$1" id="$2"
	git ls-tree -r --name-only "$ref" -- "${STATUS_PATHSPECS[@]}" 2>/dev/null |
		grep -E "/${id}(-[^/]*)?\.md\$" | head -1 || true
}

# status_of prints the status folder (open, resolved, wontfix) a ledger path sits in.
status_of() {
	local path="${1#"$ISSUES_DIR"/}"
	printf '%s\n' "${path%%/*}"
}

# frontmatter_commit prints a record's resolved_by.commit sha at ref, reading the
# frontmatter ONLY: a sha quoted in the prose body is narrative, not provenance,
# so the scan stops at the closing delimiter. RS002 and RS003 share this so the
# two rules cannot drift apart on where a stamp lives.
frontmatter_commit() {
	local ref="$1" path="$2"
	git show "$ref:$path" 2>/dev/null |
		awk 'NR>1 && /^---$/{exit} /^[[:space:]]+commit:/{print}' |
		grep -oE '[0-9a-f]{7,64}' | head -1 || true
}

# reachable reports whether sha names a real commit that ref can see. A sha that
# does not resolve at all and one that resolves but is unreachable are distinct
# faults, so they are reported separately rather than folded into "bad sha".
reachable() {
	local sha="$1" ref="$2"
	git cat-file -e "${sha}^{commit}" 2>/dev/null || return 2
	git merge-base --is-ancestor "$sha" "$ref" 2>/dev/null || return 1
	return 0
}

# declared_ids prints every iss-N an artefact DECLARES a relation to, one per
# line. The grep pair is the guard described at MENTION_RE: select the whole
# declaration lines first, then take the id out of them, so a `Resolves:` line
# mentioning a second id in a comment cannot declare it by accident.
declared_ids() {
	printf '%s\n' "$1" | grep -E "$DECLARE_RE" | grep -oE 'iss-[0-9]+' | sort -u || true
}

# mentioned_ids prints every iss-N an artefact NAMES, one per line, declarations
# included — the declaration lines are mentions too, and are cancelled by being
# matched in declared_ids rather than by being excluded here. Keeping the two
# scans independent is what makes `Refs: iss-1` + prose about iss-2 report iss-2
# alone.
mentioned_ids() {
	printf '%s\n' "$1" | grep -oE "$MENTION_RE" | grep -oE 'iss-[0-9]+' | sort -u || true
}

# check_mentions applies RS004 to one artefact: every id it NAMES must appear in
# a declaration. `declared` is passed in rather than derived, because a
# pull-request TITLE has no room for a trailer — its declaration lives in the
# body, and the two halves are judged against one declaration set.
check_mentions() {
	local label="$1" text="$2" declared="$3" id
	for id in $(mentioned_ids "$text"); do
		printf '%s\n' "$declared" | grep -qx "$id" && continue
		fail "RS004 $label names $id without declaring its relation to it. Add exactly one declaration line: 'Resolves: $id' if this change fixes it (RS001 then requires the record to enter $ISSUES_DIR/resolved/ or $ISSUES_DIR/wontfix/ in the same change), or 'Refs: $id' if it is touched but not fixed (informational; no ledger move required). Those two spellings are the whole vocabulary — 'Ref:', 'See:' and 'Related:' are not declarations."
	done
}

# canon_itd prints an intent id in the one spelling the store's filenames use —
# leading zeros trimmed, textually — or nothing for an id with no number left
# (the allocator issues no zero id). recordid.CanonCitedID is the Go original;
# `itd-007` in a trailer or a back-link names the record filed as itd-7. A value
# that is not an intent id at all — `null`, a bare number — prints nothing and so
# matches nothing, which is SameID's fail-closed half.
canon_itd() {
	case "$1" in
	[Ii][Tt][Dd]-*) ;;
	*) return 0 ;;
	esac
	local n="${1#????}"
	case "$n" in
	"" | *[!0-9]*) return 0 ;;
	esac
	n="${n#"${n%%[!0]*}"}"
	[ -n "$n" ] && printf 'itd-%s\n' "$n"
	return 0
}

# ids_entering_shipped prints, canonically, every itd-N whose record ENTERS
# shipped/ across the range: a move out of planned/ (a rename, or an add without
# rename detection) or a record filed straight into shipped/. The mirror of
# ids_entering_closed, and as there a record that only leaves planned/ enters
# nothing.
ids_entering_shipped() {
	local base="$1" head="$2"
	git diff --name-status --find-renames "$base".."$head" -- "${INTENT_PATHSPECS[@]}" |
		while IFS=$'\t' read -r status path dest; do
			local landed=""
			case "$status" in
			R*) landed="$dest" ;;
			A) landed="$path" ;;
			esac
			case "$landed" in
			"$INTENTS_DIR/shipped/"*)
				canon_itd "$(basename "$landed" | grep -oE '^itd-[0-9]+' || true)"
				;;
			esac
		done
}

# intent_path prints the store path of a canonical itd-N at ref, or nothing. The
# id is matched as a whole basename prefix, zero padding admitted, so itd-7 never
# answers for itd-70.
intent_path() {
	local ref="$1" id="$2"
	git ls-tree -r --name-only "$ref" -- "${INTENT_PATHSPECS[@]}" 2>/dev/null |
		grep -E "/itd-0*${id#itd-}(-[^/]*)?\.md\$" | head -1 || true
}

# bucket_of prints the lifecycle bucket an intent path sits in.
bucket_of() {
	local path="${1#"$INTENTS_DIR"/}"
	printf '%s\n' "${path%%/*}"
}

# frontmatter_field prints one scalar from a record's frontmatter at ref,
# unquoted, reading the frontmatter ONLY — as frontmatter_commit does, and for
# the same reason: a `intent:` line in a spec's prose is narrative.
frontmatter_field() {
	local ref="$1" path="$2" key="$3"
	git show "$ref:$path" 2>/dev/null |
		awk -v key="$key" 'NR==1{if($0!="---")exit;next} /^---$/{exit}
			index($0,key":")==1{v=substr($0,length(key)+2); gsub(/^[ \t"'"'"']+|[ \t"'"'"']+$/,"",v); print v; exit}' || true
}

# open_specs_for prints every spc-N in the spec store's open/ at ref whose own
# `intent:` back-link names the canonical itd-N. The back-link, not the intent's
# scalar spec_id, is the source of truth for which specs realise an intent
# (adr-2609151513118583): a remainder spec is named by nothing on the intent.
# An open/ holding no spec at all is an answer (none), not an error: grep's
# no-match exit (1) is accepted, or pipefail would carry it out through the
# caller's assignment and errexit would end the run with no message. Only that
# status: a git failure, or grep's own (2), still fails the pipeline, because a
# swallowed git error reads exactly like an empty store.
open_specs_for() {
	local ref="$1" id="$2" f back
	git ls-tree -r --name-only "$ref" -- "$SPECS_DIR/open" 2>/dev/null | { grep -E '\.md$' || [ "$?" -eq 1 ]; } |
		while IFS= read -r f; do
			back="$(frontmatter_field "$ref" "$f" intent)"
			[ "$(canon_itd "$back")" = "$id" ] || continue
			basename "$f" | grep -oE '^spc-[0-9]+' || true
		done
}

# check_delivery applies RS005 to one declared, canonical itd-N from commit sha.
# $shipped is the set of ids entering shipped/ in the range.
check_delivery() {
	local sha="$1" id="$2" base="$3" head="$4" shipped="$5" behind="$6"
	printf '%s\n' "$shipped" | grep -qx "$id" && return 0
	local says="RS005 commit ${sha:0:12} declares 'Delivers: $id', but"
	local head_path base_path base_bucket=""
	head_path="$(intent_path "$head" "$id")"
	base_path="$(intent_path "$base" "$id")"
	[ -n "$base_path" ] && base_bucket="$(bucket_of "$base_path")"
	if [ -z "$head_path" ] && [ -n "$base_path" ]; then
		fail "$says $id has no record at $head, while $base holds it in $INTENTS_DIR/$base_bucket/ — this branch predates the record. Rebase onto $base, then close its spec in this change (abcd spec close <spc-N>) if it is still planned there, or drop the trailer if it has already shipped."
		return 0
	elif [ -z "$head_path" ]; then
		fail "$says $id has no record at $head or at $base. Check the id — the trailer names the intent this change delivers, never its spec — or drop the trailer."
		return 0
	fi
	if [ "$base_bucket" = shipped ]; then
		# The stale-branch split RS001 draws, for the same reason: whether a rebase
		# is the remedy turns on WHEN the record reached shipped/.
		local placer
		placer="$(git log -n1 --format='%h %s' "$head".."$base" -- "$base_path" || true)"
		if [ -n "$placer" ]; then
			fail "$says $id already sits in $INTENTS_DIR/shipped/ at $base (placed there on $base's side by $placer), and $head is $behind commit(s) behind $base: the delivery reached $base outside $base..$head, so this trailer describes work $base already holds. Rebase onto $base; if this commit survives the rebase, drop the trailer."
		else
			fail "$says $id already sat in $INTENTS_DIR/shipped/ before this branch diverged from $base: the trailer names an intent delivered before this commit. Drop the trailer."
		fi
		return 0
	fi
	local bucket not_entering="$id does not enter $INTENTS_DIR/shipped/ in $base..$head"
	bucket="$(bucket_of "$head_path")"
	case "$bucket" in
	planned)
		local open_specs
		open_specs="$(open_specs_for "$head" "$id" | sort -u)"
		if [ -n "$open_specs" ]; then
			local list cmds n
			list="$(printf '%s\n' "$open_specs" | paste -sd, - | sed 's/,/, /g')"
			cmds="$(printf '%s\n' "$open_specs" | sed 's/^/abcd spec close /' | paste -sd';' - | sed 's/;/; /g')"
			n="$(printf '%s\n' "$open_specs" | grep -c .)"
			fail "$says $not_entering, and it sits in $INTENTS_DIR/planned/ with $n spec(s) still open that name it ($list). Close them in this change ($cmds) — the intent ships on the close after which no open spec names it, and the trailer means this change finishes it — or drop the trailer."
		else
			local spec_id
			spec_id="$(frontmatter_field "$head" "$head_path" spec_id)"
			case "$spec_id" in
			"" | null | "~")
				fail "$says $not_entering, and it sits in $INTENTS_DIR/planned/ with no spec to close (spec_id: null, and no open spec names it), so no close can ship it. Give it a spec first, or drop the trailer."
				;;
			*)
				fail "$says $not_entering, and it sits in $INTENTS_DIR/planned/ although no open spec names it (its spec $spec_id is closed) — the shape the release cut refuses as a stale intent. Move it to $INTENTS_DIR/shipped/ in this change, with its impact declared, or drop the trailer."
				;;
			esac
		fi
		;;
	drafts)
		fail "$says $not_entering, and it sits in $INTENTS_DIR/drafts/: it has not been planned, so it has no spec to close. Plan it (abcd intent plan $id) and close its spec in this change, or drop the trailer."
		;;
	superseded)
		fail "$says $not_entering, and it sits in $INTENTS_DIR/superseded/: a superseded intent is replaced, not delivered. Name the intent that replaced it, or drop the trailer."
		;;
	disciplines)
		fail "$says $not_entering, and it sits in $INTENTS_DIR/disciplines/: a discipline is a standing rule with no spec and no shipped state. Drop the trailer."
		;;
	*)
		fail "$says $not_entering. Close its spec in this change (abcd spec close <spc-N>) or drop the trailer."
		;;
	esac
}

check_pr() {
	local title_file="$1" body_file="$2" title body declared
	local f
	for f in "$title_file" "$body_file"; do
		[ -f "$f" ] || {
			echo "check-issue-resolution: no such file: $f" >&2
			exit 2
		}
	done
	# A body typed or edited in the forge's web UI arrives CRLF-terminated, and
	# DECLARE_RE anchors at end of line: without this, `Refs: iss-N\r` is not a
	# declaration and the gate false-reds a pull request that declared correctly.
	# check-attribution.sh's check_text normalises for the same reason.
	title="$(tr -d '\r' <"$title_file")"
	body="$(tr -d '\r' <"$body_file")"
	# The declaration set is read from BOTH halves, though only a body can
	# realistically carry a trailer line: a title that is nothing but
	# `Refs: iss-N` is a degenerate but honest declaration, and refusing it would
	# be a rule about formatting rather than about disclosure.
	declared="$(declared_ids "$title
$body")"
	# Judged separately so the refusal says WHERE the undeclared id is — the title
	# and the body are edited in different boxes.
	check_mentions "the pull-request title" "$title" "$declared"
	check_mentions "the pull-request body" "$body" "$declared"
	# Deliberately NOT fence-stripped, unlike check-attribution.sh's body arm.
	# That concession exists so a repository can DOCUMENT a banned footer shape;
	# there is no counterpart here, because a record id inside a fence is not an
	# illustration of a mention — it IS one, and the remedy costs a single
	# `Refs:` line that is true anyway. Striping would also delete declarations,
	# since a fenced commit message carries its own trailers.
	echo "check-issue-resolution: RS004 checked the pull-request title and body"
}

check_commits() {
	local base="$1" head="$2"
	local range
	range="$(git rev-list --no-merges "$base".."$head")"
	if [ -z "$range" ]; then
		echo "check-issue-resolution: no non-merge commits in $base..$head — nothing to check"
		return 0
	fi

	# A declared resolution must land the record in a terminal folder: the id must
	# ENTER resolved/ or wontfix/ in this range. That is the honest test of the
	# trailer — the changelog derives from records reaching a terminal folder — and
	# it holds both for a record moved out of open/ AND for one captured and
	# resolved in the same change (a two-dot diff shows the latter only as an add
	# into resolved/, never as a departure from open/). A bare `git rm` of the open
	# record enters nothing, so it does NOT satisfy the trailer: the record would
	# otherwise vanish from the ledger, its changelog line lost, with no other gate
	# to catch it. (A record that enters resolved/ while a copy stays in open/ is a
	# duplicate id, which record-lint's issue_id_unique refuses.)
	local closed
	closed="$(ids_entering_closed "$base" "$head" | sort -u)"

	# RS001 — a declared resolution must move the record. Every shape below is a
	# refusal; they differ in the diagnosis, and the diagnosis is what a reader
	# acts on. On a branch 235 commits behind main this rule emitted 84
	# violations that each said "resolve it in this change" about a record the
	# base already held in resolved/ — the branch's own work, squash-merged, so
	# both trees held the moved record and the two-dot diff showed it entering
	# nothing. The one remedy was a rebase, which no line named
	# (iss-2609012023256534). The refusal stands in that shape: a rebase makes
	# the range honest, and the merged commits vanish from it. What changes is
	# that the message now says what the script can prove.
	local shipped
	shipped="$(ids_entering_shipped "$base" "$head" | sort -u)"

	local declared="" delivered=""
	local behind
	behind="$(git rev-list --count "$head".."$base")"
	local scanned=0
	while IFS= read -r sha; do
		[ -n "$sha" ] || continue
		# RS004 — every id this message names must declare its relation. It reads
		# the same $range as RS001, so MERGE COMMITS ARE EXEMPT: `Merge pull
		# request #N from …` is composed by the forge, and a merge's body can carry
		# a branch's text that was already judged, commit by commit, on the branch.
		# A SQUASH merge is not a merge commit — its message is the branch's
		# messages concatenated, so it arrives here carrying the branch's own
		# declarations and passes for the same reason the branch did.
		local msg
		msg="$(git show -s --format='%B' "$sha")"
		check_mentions "commit ${sha:0:12}" "$msg" "$(declared_ids "$msg")"
		scanned=$((scanned + 1))
		while IFS= read -r line; do
			# RS005 — a declared delivery must ship the intent. Judged on the same
			# lines RS001 reads; a line is one trailer or the other, never both.
			if [[ "$line" =~ $DELIVERS_LOOSE_RE ]] && [[ "${line#*:}" =~ $DELIVERS_ID_SHAPED_RE ]]; then
				local raw canon_ok=1
				if [[ "$line" =~ $DELIVERS_RE ]]; then
					for raw in $(printf '%s\n' "$line" | grep -oE 'itd-[0-9]+'); do
						[ -n "$(canon_itd "$raw")" ] || canon_ok=0
					done
				else
					canon_ok=0
				fi
				if [ "$canon_ok" -eq 0 ]; then
					fail "RS005 commit ${sha:0:12} carries a delivery line RS005 cannot read: '$line'. The trailer is spelled exactly 'Delivers: itd-N' (a comma-separated list of intent ids is allowed) and names the intent this change finishes — never its spec, which is closed with abcd spec close."
					continue
				fi
				for raw in $(printf '%s\n' "$line" | grep -oE 'itd-[0-9]+'); do
					local cid
					cid="$(canon_itd "$raw")"
					delivered="$delivered $cid"
					check_delivery "$sha" "$cid" "$base" "$head" "$shipped" "$behind"
				done
				continue
			fi
			[[ "$line" =~ $TRAILER_RE ]] || continue
			# Every id on the line, not just the first: a `Resolves:` list declares
			# a resolution for each of them, and an id RS001 did not read would be
			# a declared resolution with no move requirement behind it — the exact
			# drift this rule exists to stop, reopened by a comma.
			local id
			for id in $(printf '%s\n' "$line" | grep -oE 'iss-[0-9]+'); do
				declared="$declared $id"
				printf '%s\n' "$closed" | grep -qx "$id" && continue
				local head_path base_path base_status
				head_path="$(record_path "$head" "$id")"
				base_path="$(record_path "$base" "$id")"
				base_status=""
				[ -n "$base_path" ] && base_status="$(status_of "$base_path")"
				# Absence from the head tree is the most specific fact and is
				# checked first: whatever the base holds, "resolve it in this
				# change" cannot be done for a record the tree lacks.
				if [ -z "$head_path" ] && [ -n "$base_path" ]; then
					fail "RS001 commit ${sha:0:12} declares 'Resolves: $id', but $id has no record at $head, while $base holds it in $ISSUES_DIR/$base_status/ — this branch predates the record. Rebase onto $base, then resolve it in this change (abcd capture resolve $id ...) if it is still open there, or drop the trailer if it is already terminal."
					continue
				elif [ -z "$head_path" ]; then
					fail "RS001 commit ${sha:0:12} declares 'Resolves: $id', but $id has no record at $head or at $base. Check the id, or capture the issue and resolve it in this change (abcd capture resolve $id ...)."
					continue
				fi
				case "$base_status" in
				resolved | wontfix)
					# Terminal at the base. Whether a rebase is the remedy turns on
					# WHEN it got there: a base-side commit the head lacks placed it
					# after the branch diverged (the stale-branch shape), or it was
					# terminal already at the merge base, in which case the trailer
					# names an issue resolved before this commit and nothing but
					# dropping it helps. The behind-count alone cannot tell them apart;
					# the record's base-side history can.
					local placer
					placer="$(git log -n1 --format='%h %s' "$head".."$base" -- "$base_path" || true)"
					if [ -n "$placer" ]; then
						fail "RS001 commit ${sha:0:12} declares 'Resolves: $id', but $id already sits in $ISSUES_DIR/$base_status/ at $base (placed there on $base's side by $placer), and $head is $behind commit(s) behind $base: the resolution reached $base outside $base..$head, so this trailer describes work $base already holds. Rebase onto $base; if this commit survives the rebase, drop the trailer."
					else
						fail "RS001 commit ${sha:0:12} declares 'Resolves: $id', but $id already sat in $ISSUES_DIR/$base_status/ before this branch diverged from $base: the trailer names an issue that was resolved before this commit. Drop the trailer."
					fi
					;;
				*)
					fail "RS001 commit ${sha:0:12} declares 'Resolves: $id', but $id does not enter $ISSUES_DIR/resolved/ or $ISSUES_DIR/wontfix/ in $base..$head. Resolve it in this change (abcd capture resolve $id ...) or drop the trailer."
					;;
				esac
			done
		done <<<"$(git show -s --format='%B' "$sha")"
	done <<<"$range"

	# RS002 — a stamp added here must name a commit this head can see. Read the
	# frontmatter of each record the range touched and check only a resolved_by.commit
	# this range introduced or changed. Scanning the raw diff for `+  commit:` instead
	# would reachability-check a `commit:` example in a record's prose body — a false
	# violation — which is exactly the boundary RS003 already draws.
	# rc-checked like the ledger listing: the || true belongs to grep's no-match
	# exit alone, never to a git failure.
	local diffout changed
	local rc=0
	diffout="$(git diff --name-only "$base".."$head" -- "${STATUS_PATHSPECS[@]}" 2>&1)" || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "check-issue-resolution: git diff failed for $base..$head (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
		echo "$diffout" >&2
		exit 2
	fi
	changed="$(printf '%s\n' "$diffout" | grep -E '\.md$' || true)"
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		local head_sha base_sha
		head_sha="$(frontmatter_commit "$head" "$f")"
		[ -n "$head_sha" ] || continue
		base_sha="$(frontmatter_commit "$base" "$f")"
		[ "$head_sha" = "$base_sha" ] && continue
		local rc=0
		reachable "$head_sha" "$head" || rc=$?
		case "$rc" in
		2) fail "RS002 resolved_by.commit '$head_sha' is added in this range but names no commit in this repository. --commit is shape-checked only, so a wrong sha is accepted silently." ;;
		1) fail "RS002 resolved_by.commit '$head_sha' is not reachable from $head. Cite a commit on this branch or already on main." ;;
		esac
	done <<<"$changed"

	# RS006 — a resolution that names a test names one that exists. The note is
	# what every later reader trusts about how a fix was proved, and one was
	# false: it named three tests for a guard none of them exercised
	# (iss-2609020716579024). This is the cheap rung: each TestX the resolution
	# field names must be defined by some _test.go file at head. Only records
	# entering a terminal folder in this range are read, because a test renamed
	# long after a record was closed does not make its note false when written.
	local rs006=0
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		local rpath note name
		rpath="$(record_path "$head" "$id")"
		[ -n "$rpath" ] || continue
		note="$(git show "$head:$rpath" 2>/dev/null | awk 'NR>1 && /^---$/{exit} /^resolution:/{print}')"
		while IFS= read -r name; do
			[ -n "$name" ] || continue
			rs006=$((rs006 + 1))
			if ! git grep -qE "^func ${name}\(" "$head" -- '*_test.go' 2>/dev/null; then
				fail "RS006 $id's resolution names $name, which no _test.go file at $head defines. A resolution note is what later readers trust about how the fix was proved; name the test that exists, or say what proves the fix without naming one."
			fi
		done <<<"$(printf '%s\n' "$note" | grep -oE 'Test[A-Z][A-Za-z0-9_]*' | sort -u || true)"
	done <<<"$closed"

	if [ -n "${declared// /}" ]; then
		echo "check-issue-resolution: RS001 checked$declared"
	fi
	echo "check-issue-resolution: RS006 checked $rs006 test name(s) in resolutions entering a terminal folder"
	if [ -n "${delivered// /}" ]; then
		echo "check-issue-resolution: RS005 checked$delivered"
	fi
	echo "check-issue-resolution: RS004 checked $scanned commit message(s) for undeclared record mentions"
}

check_ledger() {
	local ref="${1:-HEAD}"
	local checked=0
	# The listing probe is rc-checked so a git failure exits 2 as an environment
	# fault; only a git success with zero matches is the legitimate empty-ledger
	# pass, and it stays loud so it cannot be mistaken for a verdict.
	local listing files
	local rc=0
	listing="$(git ls-tree -r --name-only "$ref" -- "${STATUS_PATHSPECS[@]}" 2>&1)" || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "check-issue-resolution: git ls-tree failed at $ref (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
		echo "$listing" >&2
		exit 2
	fi
	files="$(printf '%s\n' "$listing" | grep -E '\.md$' || true)"
	[ -n "$files" ] || {
		echo "check-issue-resolution: no ledger records at $ref — nothing to check"
		return 0
	}
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		# Only the frontmatter's resolved_by block carries a stamp; a sha quoted in
		# the prose body is narrative, not provenance, so the scan stops at the
		# closing delimiter (frontmatter_commit).
		local sha
		sha="$(frontmatter_commit "$ref" "$f")"
		[ -n "$sha" ] || continue
		checked=$((checked + 1))
		local rc=0
		reachable "$sha" "$ref" || rc=$?
		case "$rc" in
		2) fail "RS003 $f stamps resolved_by.commit '$sha', which names no commit in this repository." ;;
		1) fail "RS003 $f stamps resolved_by.commit '$sha', which is no longer reachable from $ref — a squash or rebase merge rewrote it." ;;
		esac
	done <<<"$files"
	echo "check-issue-resolution: RS003 checked $checked stamped record(s) at $ref"
}

# A shallow checkout cannot run these checks honestly: reachable() cannot tell
# an absent commit from an unfetched one, so RS002/RS003 would refuse every
# stamp whose commit lies past the graft — 85 false violations on a clean tree,
# each carrying a diagnosis about a repository fault that did not happen. The
# spec that ruled git resolution out of --commit (spc-25) names shallow states
# in-envelope, so the environment fault must be reported as itself: exit 2, the
# code the contract reserves for it, never a violation.
# The probe itself is rc-checked: comparing a failed substitution against
# "true" would let the very fault this arm exists to report disarm it.
rc=0
shallow="$(git rev-parse --is-shallow-repository 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
	echo "check-issue-resolution: git rev-parse --is-shallow-repository failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
	echo "$shallow" >&2
	exit 2
fi
case "$shallow" in
true)
	echo "check-issue-resolution: shallow checkout — RS002/RS003 cannot tell an absent commit from an unfetched one; run 'git fetch --unshallow' first (CI checks out with fetch-depth: 0)." >&2
	exit 2
	;;
false) ;;
*)
	echo "check-issue-resolution: unexpected git rev-parse --is-shallow-repository output \"$shallow\" — refusing rather than guessing." >&2
	exit 2
	;;
esac

case "${1:-}" in
commits)
	[ $# -eq 3 ] || usage
	check_commits "$2" "$3"
	;;
ledger)
	[ $# -le 2 ] || usage
	check_ledger "${2:-HEAD}"
	;;
pr)
	# RS004 over the artefact no commit message reaches: the title and body a
	# human types into the forge. Two FILES rather than two arguments, because
	# both are attacker-controlled text and a workflow that spliced them into an
	# argv would be the template-injection hole check-attribution.sh's body arm
	# avoids the same way.
	[ $# -eq 3 ] || usage
	check_pr "$2" "$3"
	;;
*)
	usage
	;;
esac

if [ "$violations" -gt 0 ]; then
	printf 'check-issue-resolution: FAILED — %d violation(s)\n' "$violations" >&2
	exit 1
fi
echo "check-issue-resolution: OK"
