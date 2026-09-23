---
id: itd-2609111003026787
slug: a-gate-catches-the-intent-whose-work-is-live-while-its-spec
spec_id: spc-2609120450289528
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_adrs: []
severity: major
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# A gate catches the intent whose work is live while its spec is still open

_Typed links: promoted from `iss-2609091642508005`, which states this remedy in
its own words and carries the measured population. The verbs this record first
proposed are held until the gate has run a cycle, on the maintainer's ruling of
2026-09-11._

## Press Release

An open-source maintainer lands the work an intent promised, opens the pull
request, and forgets the one step nothing runs for them: closing the spec. Today
that intent stays in `planned/` indefinitely. It ships with no changelog line,
the release exits 0, and nobody finds out until somebody reads.

With this gate they find out while the change is still in their hands. The
commit already says which intent it delivers, the way it already says which issue
it resolves; the gate reads that line and refuses to let the change land with the
intent still planned, naming the command that closes it.

"I do not need a new verb. I need to be told, at the moment it matters, that I
forgot the one I already had," said Kira, an open-source maintainer.

## Why This Matters

The population is measured, not estimated: **61 planned intents, every one with
its spec still open** (`iss-2609091642508005`). At least one is a confirmed live
instance — `itd-121`'s record-id dispatch has been shipping behaviour while
`spc-26` sat in `open/`, and it was found by a person reading, not by a gate.

Half of this hole is already closed, which is what makes the other half worth
closing. `release/emit.go:294-332` (`staleIntents`, `RefusalStaleIntent`) refuses
a cut carrying a planned intent whose spec has **closed**, naming each one. The
symmetric case — spec still **open** — is invisible, and it is the more common
one, because closing the spec is exactly the step that gets forgotten.

This record was filed proposing verbs instead, and two earlier rulings had
already declined them. `iss-2609011026401952` resolved the same complaint with
"No new verb was needed", and left the alternative open in its grounds: "a
mechanical gate that refuses a merged intent whose spec is still open would be
the durable fix". `iss-2609091642508005` then measured the population and said
what is owed is "the detector, not another audit by hand". The `spc-26` case
argues the same thing from the other end: the verb existed, was one command
away, and went unrun. A second verb is a second verb nobody runs.

## Prior Art

- **`iss-2609091642508005`** (open, `major`, deferred after v0.7.1) — the record
  this is promoted from. States the remedy, carries the measured population of
  61 and the two confirmed instances.
- **`iss-2609011026401952`** (resolved) — the same complaint resolved against a
  verb, naming the mechanical gate as the durable fix. This record builds what
  that ruling left open rather than reversing it.
- **`release/emit.go:294-332`** — `staleIntents` and `RefusalStaleIntent`: the
  closed-spec half of this gate, already shipped. The pattern to follow, and the
  reason the new refusal should read like an existing one rather than a new
  dialect.
- **`internal/core/lint/lint.go:2199-2252`** — `checkSpecLifecycle` checks id,
  slug, link existence and bidirectional agreement, and never bucket coherence
  between the two stores. The nearest existing home, and evidence the check is
  absent rather than merely weak.
- **`itd-34`** (planned) — owns intent-kind taxonomy and the supersede path. Not
  related to this record; named here because an earlier revision wrongly absorbed
  its supersession clause, and the absorption was reverted.
- **The retirement verbs** — `intent ship` and the intent supersede path remain
  unowned and unfiled. Held deliberately: the maintainer ruled on 2026-09-11 that
  the gate runs for a cycle first, so that what people actually get wrong is
  measured before a verb is designed for it.

## What's In Scope

- A mechanical refusal for the state: intent in `planned/`, its work live, its
  spec still `open/`.
- A refusal that reads like `RefusalStaleIntent` and names every offending
  record and the command that fixes each.
- The signal, settled at interview: a commit that delivers an intent SAYS SO in
  its trailer, and the gate refuses when the named intent is not shipped in the
  same change. No liveness is inferred; the author declares it. This is the shape
  `Resolves: iss-N` and RS001 already have for issues.

## What's Out of Scope

- **`intent ship` and the intent supersede verb.** Held for a cycle by ruling.
- **An ADR supersede verb.** Five lifetime invocations, two legal dispositions
  one of which prunes the file, no directory move, and `schema.go:606-645`
  already refuses a one-way edge as a blocker.
- **Unifying the retirement vocabulary.** The four words carry distinctions that
  a shared vocabulary would flatten, and the per-family impact vocabularies
  differ on purpose (`intent/lifecycle.go:691-700`).
- **Backfilling the 61.** Deciding what to do with the existing population is
  its own act; this record makes the state visible.

## Mechanism

We expect a refusal to close this gap because the remedy it replaces was
documentation, and documentation cannot refuse. The 61 records accumulated under
a rule every contributor had already read, stated in the definition of done and
repeated on the surface pages; the thing that was missing was never the wording.
A gate that stops the change is a different kind of object from a rule that
describes it.

What would show this wrong: authors omitting the trailer in order to avoid the
refusal. That would make the gate a thing to route around rather than a thing
that helps, and it would show the cause was not forgetting but reluctance — in
which case the remedy is somewhere else entirely, because a gate keyed on a
voluntary declaration cannot reach someone who declines to declare.

## Scope Conditions

- The people using abcd mean to keep the convention and forget the last step, <!-- cond: cond-2609120450285738 -->
  rather than skipping it on purpose. The gate reads a line the author chose to
  write, so it only ever helps the willing: a larger contributor base, or one
  that finds the rule irksome, simply omits the line and the gate never fires.
- The refusal reaches the author while the change is still theirs, before it <!-- cond: cond-2609120450285806 -->
  merges. The same check at release time would interrupt whoever happens to be
  cutting the release about somebody else's forgotten step, days late — the same
  rule at a moment where acting on it is somebody else's work.
- A piece of work is finished by a single change. Where an intent arrives across <!-- cond: cond-2609120450285032 -->
  several pull requests, no one change can honestly say it finishes it, so the
  gate either objects to the first partial delivery or teaches authors to stop
  writing the line.
- Closing a spec stays one quick command with no ceremony. If it later needs a <!-- cond: cond-2609120450283625 -->
  review sign-off or an audit first, being told "you forgot to close it" blocks
  the author on somebody else's availability, and avoiding the trailer becomes
  cheaper than triggering the check.

## Acceptance Criteria

> _Confirmed by the maintainer at the planning interview of 2026-09-11. The set
> was walked bullet by bullet; five were accepted as written and the fifth was
> rewritten on the maintainer's instruction, because "reads as one convention"
> was a review judgement wearing Given-When-Then clothes and could not fail._

- **Given** a change whose commit declares that it delivers an intent, **when**
  that intent does not reach `shipped/` in the same change, **then** the gate
  refuses and names the command that closes its spec.
- **Given** a change that declares delivery of several intents, **when** any one
  of them stays planned, **then** every unshipped one is named, so the author
  clears them in one pass rather than one refusal at a time.
- **Given** a change that declares no delivery, **when** it lands, **then**
  nothing is refused, because a planned intent nobody claims to have built is the
  ordinary state of the backlog.
- **Given** a change that declares delivery of an intent that does not exist, or
  one already shipped, **when** the gate runs, **then** it says which, rather
  than passing silently or refusing without naming the cause.
- **Given** the existing issue rule's refusal, **when** the intent rule refuses,
  **then** it uses the same message shape and the same exit code, and a test
  compares the two rather than a reviewer judging that they feel alike.
- **Given** the 61 records already planned with open specs, **when** the gate
  arms, **then** none of them is refused by it, because none declares a trailer —
  and the record says plainly that clearing them is a separate act.

## Open Questions

- **RESOLVED at interview 2026-09-11: the signal is a commit trailer.** Nothing
  in a record says an intent's work has landed — specs carry no task checkboxes
  and intents carry no delivering-commit stamp — so no liveness can honestly be
  inferred. The author declares it instead, and the gate enforces the
  declaration. Open underneath it: the trailer's spelling, and whether it takes
  the intent id or the spec id.
- **What the trailer gate does NOT catch, and whether that is acceptable.** It
  fires only on a change that declares the trailer. A change that delivers an
  intent silently is invisible to it, and so are the 61 records already in this
  state — which arrived there by nobody saying anything, which is the same
  absence. The issue rule accepts exactly this limit ("Resolving without a
  trailer stays legal"), and the argument for accepting it here is the same: the
  trailer is cheap to write while describing the change, and the gate converts a
  description a person was going to write anyway into an enforced close. The
  argument against is that the measured problem is silence, and a gate keyed on
  speech does not reach it.
- **Where does the gate live?** The trailer decides this largely: RS001 runs at
  merge time in `lint-issues`, so the symmetric rule belongs beside it rather
  than at the cut. `staleIntents` stays where it is and keeps the closed-spec
  half at release time.
- **What happens to the 61?** The trailer gate does not reach them, so they are
  not a wedge risk — and they are also not fixed. Clearing them is a separate
  act, and this record should say whether it owns it or hands it back to
  `iss-2609091642508005`.
- **Is a refused release the right severity?** A merge-time finding is cheaper to
  act on and harder to ignore; a release-time refusal is louder and later.

## Audit Notes


_None yet: the intent has not shipped._

<!-- abcd-review: INGESTED receipt=rcp-600cc27b0c43 -->
Fidelity review — receipt rcp-600cc27b0c43 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:eb995a6229933334408bd589e5d78f4bf0f3a96f316cc1a0d3bee7ad482e769d
Input attestations: diff:8486c141..97e8ce1a (PR #667, merged 0c0ca52d; tree read at cede78b8)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: a Delivers: id absent from the set entering shipped/ is a refusal, and the planned-with-open-spec branch names `abcd spec close <spc-N>` for each open spec; the cases script asserts the refusal text down to the close command
  evidence: scripts/check-issue-resolution.sh:415 — "printf '%s\n' "$shipped" | grep -qx "$id" && return 0"
  evidence: scripts/check-issue-resolution.sh:451 — "Close them in this change ($cmds)"
  evidence: scripts/check-issue-resolution-cases.sh:792 — "expect_refusal_naming "$d" "RS005 trailer, intent left in planned/ with its spec open""
- ac-2 — MET: every id on every Delivers: line is judged by check_delivery in turn, and the cases assert three unshipped intents across two trailer lines are each named and counted as their own violation
  evidence: scripts/check-issue-resolution.sh:586 — "check_delivery "$sha" "$cid" "$base" "$head" "$shipped" "$behind""
  evidence: scripts/check-issue-resolution-cases.sh:826 — "Delivers: itd-7, itd-8"
  evidence: scripts/check-issue-resolution-cases.sh:834 — "expect_refusal_naming "$d" "RS005 counts every unshipped intent as its own violation""
- ac-3 — MET: the rule is keyed on the trailer, and a case commits an edit to a planned record with open specs all around and no trailer, expecting a pass
  evidence: scripts/check-issue-resolution.sh:216 — "change that declares no delivery is refused nothing (the intent's criterion"
  evidence: scripts/check-issue-resolution-cases.sh:844 — "expect pass "$d" "RS005 no trailer, planned intents with open specs are not refused""
- ac-4 — MET: an id with no record at head or base, one already in shipped/ before the branch, and one shipped on the base side after divergence are three distinct refusals, each case-held and each asserted not to prescribe the wrong remedy
  evidence: scripts/check-issue-resolution.sh:425 — "fail "$says $id has no record at $head or at $base."
  evidence: scripts/check-issue-resolution.sh:436 — "the trailer names an intent delivered before this commit. Drop the trailer.""
  evidence: scripts/check-issue-resolution-cases.sh:853 — "expect_refusal_naming "$d" "RS005 on an id with no record anywhere says so""
  evidence: scripts/check-issue-resolution-cases.sh:864 — "expect_refusal_naming "$d" "RS005 on an intent already shipped before the branch says to drop the trailer""
- ac-5 — MET: RS005 reuses RS001's fail helper and exit path, and the cases script normalises one RS001 and one RS005 refusal through a single pattern and requires them byte-identical, exit code included
  evidence: scripts/check-issue-resolution.sh:224 — "fail() {"
  evidence: scripts/check-issue-resolution-cases.sh:1042 — "# Criterion 5: the intent rule's refusal has the issue rule's shape and exit"
  evidence: scripts/check-issue-resolution-cases.sh:1068 — "if [ "$shape_iss" != "$shape_itd" ]; then"
  evidence: scripts/check-issue-resolution-cases.sh:1071 — "grep -qx 'exit=1'"
- ac-6 — MET: the gate is armed in CI and preflight over the PR range, fires only on a trailer, and the record states in both the intent and the spec that clearing the standing backlog is a separate act handed to its own issue
  evidence: .github/workflows/ci.yml:473 — "bash scripts/check-issue-resolution.sh commits "$BASE_SHA" HEAD"
  evidence: Makefile:183 — "@bash scripts/check-issue-resolution.sh commits origin/main HEAD"
  evidence: scripts/check-issue-resolution-cases.sh:837 — "# Criterion 3 and 6: no trailer, no refusal — even with planned intents whose"
  evidence: .abcd/development/specs/closed/spc-2609120450289528-a-gate-catches-the-intent-whose-work-is-live-while-its-spec.md:29 — "Clearing the standing 61, which this gate"

Gap audit:
- honoured:
  - one trailer, one rule, beside RS001 at merge time
    evidence: scripts/check-issue-resolution.sh:208 — "DELIVERS_RE='^Delivers:[[:space:]]+itd-[0-9]+([[:space:]]*,[[:space:]]*itd-[0-9]+)*[[:space:]]*$'"
    evidence: .github/workflows/ci.yml:465 — "Declared resolutions and deliveries move the record, mentions declare themselves (RS001/RS002/RS004/RS005)"
  - a planned intent with spec_id: null is told it has no spec to close rather than a command that cannot run
    evidence: scripts/check-issue-resolution.sh:457 — "with no spec to close (spec_id: null, and no open spec names it), so no close can ship it"
  - the trailer means the change finishes the intent, and the contributor guide says so
    evidence: CONTRIBUTING.md:67 — "`Delivers: itd-N` for an intent the change **finishes** — not one it"
  - the gate reads the intent and spec stores from declared paths
    evidence: internal/core/lint/preflightgates_test.go:340 — "func TestIssueResolutionGateReadsTheIntentAndSpecStores"
- diverged:
  - the trailer is matched exactly as RS001 matches its own: delivered wider, refusing a near-miss spelling that carries an id-shaped token rather than passing it over
    evidence: scripts/check-issue-resolution.sh:579 — "carries a delivery line RS005 cannot read"
    evidence: scripts/check-issue-resolution-cases.sh:994 — "for spelling in "Delivers: spc-7" "delivers: itd-7" "Delivers: itd-7 and itd-8"; do"
  - the ordinary refusal names the intent's spec_id: delivered as every open spec whose back-link names the intent, per the 1:n rule
    evidence: scripts/check-issue-resolution.sh:451 — "with $n spec(s) still open that name it ($list)"
    evidence: scripts/check-issue-resolution-cases.sh:946 — "expect_refusal_naming "$d" "RS005 names the remainder spec still open""
- missing: (none)

Scope-condition dispositions:
- cond-2609120450285738 — untested: whether authors write the trailer or omit it to dodge the gate is what the cycle after arming measures; nothing in the delivered diff exercises it
- cond-2609120450285806 — survived: the rule runs in the record-lint job on the pull request's base sha and on the merge-queue entry, and in preflight against origin/main, so the refusal reaches the author before the merge
  evidence: .github/workflows/ci.yml:467 — "BASE_SHA: ${{ github.event.pull_request.base.sha || github.event.merge_group.base_sha || github.event.before }}"
  evidence: Makefile:183 — "@bash scripts/check-issue-resolution.sh commits origin/main HEAD"
- cond-2609120450285032 — survived: the trailer is defined as the finishing change's, the guide tells a multi-PR author to write it on the last one, and the gate judges the whole range so a close in a later commit of the same change passes
  evidence: CONTRIBUTING.md:68 — "contributes to, since an intent that arrives across several pull requests is"
  evidence: scripts/check-issue-resolution-cases.sh:815 — "expect pass "$d" "RS005 trailer with the close in a later commit of the range""
- cond-2609120450283625 — survived: closing a spec is still one command with no sign-off, and the refusal names exactly that command
  evidence: commands/intent.md:378 — "spec close < spc-N> --json # open/ -> closed/, and planned/ -> shipped/ when it was the last open spec"
  evidence: scripts/check-issue-resolution.sh:475 — "Close its spec in this change (abcd spec close < spc-N>) or drop the trailer."
## Grounds

- pursued: we expect the gate to measure what people actually get wrong, and that measurement is what decides whether the retirement verbs are worth building at all — which is why it was chosen ahead of them rather than beside them. What would show this wrong is a cycle of refusals that cluster on something neither the gate nor a verb addresses, meaning the forgotten close was a symptom and we treated it as the disease.
