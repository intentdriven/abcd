---
id: adr-2609151513118583
slug: an-intent-owns-one-or-more-specs-and-it-ships-when-its-last
status: accepted
date: 2026-09-15
supersedes: null
superseded_by: null
related_intents: [itd-80]
related_rfcs: []
related_adrs: [adr-26, adr-31]
---

# ADR-2609151513118583: An intent owns one or more specs, and it ships when its last spec closes

## Context

Closing a spec ships its intent, unconditionally, and there is no third option.
`abcd spec close <spc-N>` is one operation with two halves: `intent.Reconcile`
(`internal/core/intent/lifecycle.go`) moves the linked intent `planned/` →
`shipped/` and then `spec.Close` (`internal/core/spec/store.go`) moves the spec
`open/` → `closed/`. The ordering is intent-first so a partial failure retries
cleanly, and the coupling is not accidental: it is what
[itd-80](../../intents/shipped/itd-80-intent-lifecycle-automation.md)
shipped, on the reasoning that the step which happens after the merge is the
step that gets forgotten.

The case it cannot express arrived in the field. In an autonomous run over a
managed repository a spec was complete and ready to close while the intent it
realised had roughly half its acceptance criteria met. The session stopped and
asked rather than close, which was the right instinct and is why the record
exists:
[iss-2609100508566552](../../../work/issues/resolved/iss-2609100508566552-spec-close-ships-the-intent-with-no-way-to-split-the-two.md).
The verb offered no way to close the spec without shipping the intent, and no
way to let the delivered part land while the rest stayed planned. Whatever the
session did next would have been a false record: ship an intent whose criteria
are half unmet, or leave finished work uncommitted to the ledger.

A shipped intent with unmet criteria is the false-green shape at the level of
the record, and it is expensive in three places at once. The release cut derives
its version and its changelog section from what sits in the terminal folders
(`internal/core/changelog/shipped.go`, per
[adr-31](0031-derived-versioning-from-intents.md)), so the cut announces the
whole intent. The fidelity audit that would catch the gap runs at the ship
transition and is owed rather than performed. And the `shipped/` bucket stops
being a fact about delivery and becomes a claim the tool asserted on the
operator's behalf.

The adjacent finding is the same seam from the other side.
[iss-2609091732329046](../../../work/issues/resolved/iss-2609091732329046-closing-a-spec-moves-its-intent-but-leaves-every-link-that-n.md)
reports that the close moves the intent and leaves every link written against
the intent's old folder pointing at nothing — three closes in one sitting
produced eight dead links and a red gate immediately afterwards. Both records
are about a close that does more than close, silently. This record rules the
lifecycle question; the link question is a defect in the same verb and is not
decided here.

Four constraints were already locked before this decision. The native spec store
is the scheduling home for a committed capability
([adr-26](0026-native-spec-layer-ccpm-backend.md)). Terminal folder membership
is what the release reads, so nothing user-facing is announced until a record
reaches one ([adr-31](0031-derived-versioning-from-intents.md)). `shipped/` is
the one bucket the record gate requires an `impact` judgement in, and
`resolveShipImpact` runs ahead of every write for the reason
[iss-126](../../../work/issues/resolved/iss-126-intent-reconcile-cannot-stamp-impact.md)
gives — abcd must not produce, out of its own verbs alone, a record its own
record-lint refuses. And an intent's acceptance criteria are the verifiable bar
for "shipped", per the [itd-1](../../intents/disciplines/itd-1-acceptance-gates.md)
discipline.

What the tree cannot currently represent is the shape the field case needs. The
link is 1:1 on both sides: an intent carries a scalar `spec_id`, a spec carries a
scalar `intent`, and `Reconcile` refuses outright when more than one spec claims
one intent — "link ambiguous — N specs realise itd-X". So a second spec for the
remainder is not merely unsupported today; minting one makes the close of the
first refuse.

## Decision

**An intent owns one or more specs.** The relation between an intent and the
specs that realise it is 1:n. One spec remains the ordinary case and needs no
ceremony; a second exists whenever delivery did not fit in one piece of
scheduled work.

**An intent is a thought-through statement of a capability, and delivery never
rewrites it.** Where a spec delivers part of an intent, the intent stands as
written. Narrowing its acceptance criteria to match what was built is not
available: it would make the record agree with the delivery by falsifying the
thing the product thinker decided.

**A spec that delivers part of an intent is closed on its own terms, and a new
spec is minted for the remainder and attached to the same intent.** The visible
state after that operation is exactly what happened: spec closed X, spec open Y,
intent still `planned/`.

**Closing a spec never ships an intent that still has an open spec.** The close
of a spec is a statement about that spec's work and about nothing else.

**The intent ships when its last spec closes.** That transition — the close
after which the intent has no open spec left — is the one that moves it
`planned/` → `shipped/`, and it stays automatic. Nothing is shipped by hand.

**`--impact` is still required at the ship transition, and only there.** The
judgement is what `shipped/` requires and what the derived version reads, so the
close that ships the intent demands it exactly as today; an earlier close, which
ships nothing, must not ask for it.

**More than one spec naming one intent is the normal state, not an ambiguity.**
The guard that refuses it is refusing the shape this record establishes, and it
is replaced by the question that actually matters: does this intent have an open
spec left?

## Alternatives Considered

1. **Close still means ship, and the intent is narrowed to what was
   delivered.** The cheapest reconciliation, and the one that needs no code: edit
   the acceptance criteria down so the shipped record is true of what was built,
   and file the rest as a new intent later. Rejected, and this is the one the
   product thinker ruled out by name. An intent that has been thought through is
   the statement of the capability; rewriting it at delivery time makes the
   record agree with whatever happened, which is the opposite of a bar. It also
   destroys the evidence: after the narrowing, nothing in the tree says the
   intent was ever wider, so the missing half is not deferred work but a thing
   nobody can find.

2. **A spec may close without shipping, and the intent is shipped by hand.** A
   `--no-ship` flag on the close, or a separate ship verb the operator runs when
   the intent is genuinely done. Rejected: it is the shape this project has
   repeatedly found to fail, because the step that happens after the merge is the
   step that gets forgotten. It also moves the shipped claim out of the record
   and into somebody's memory of what they meant to do, and an intent left in
   `planned/` with all its specs closed is invisible to the release cut, which
   then under-bumps and announces nothing.

3. **Split the intent at close: mint a successor intent for the unmet criteria
   and ship the delivered half.** The source record's own second suggestion, and
   attractive because it needs no new cardinality — every intent keeps one spec.
   Rejected on the same ground as option 1, one level up. The press release is
   one user moment; cutting it at the boundary where delivery happened to stop
   produces two records, each describing half a moment, and the split is made by
   the accident of what fitted in a cycle rather than by a decision about the
   product. A thought-through intent should not have to be re-thought because a
   cycle ran out.

4. **Keep the 1:1 coupling, but refuse the close when the intent's criteria are
   visibly unmet.** The source record's fallback: at minimum, do not move it
   silently. Rejected as a remedy on its own, because nothing in the system can
   read acceptance criteria and decide they are met — that judgement is the
   fidelity audit's, and it runs after the ship. A refusal that cannot evaluate
   its own condition degrades to a prompt on every close, which is a step people
   learn to clear rather than read. The honesty it asks for is real; it is
   delivered here by making the true state representable instead.

5. **Chosen: an intent owns one or more specs; a partial delivery closes its
   spec and mints another for the remainder; the intent ships when its last spec
   closes.** It leaves the intent intact, keeps the ship transition automatic and
   derived rather than hand-asserted, and makes the state the field case was in —
   work finished, capability not yet whole — a state the record can hold instead
   of a dilemma the session has to resolve by lying in one direction or the
   other.

## Consequences

- **`spec close` gains one question and loses one refusal.** After closing the
  spec, it asks whether any other spec naming that intent is still open. If one
  is, the intent stays in `planned/`, the verb says so and names the open spec;
  if none is, the intent ships exactly as today, with `--impact` resolved ahead
  of the move. The ambiguity guard in `intent.Reconcile` — which today refuses
  when more than one spec claims an intent — inverts: the set of claiming specs
  stops being an error and becomes the input to the question.
- **The record schema has no field for this today, and something has to carry
  it.** An intent carries a scalar `spec_id` and a spec carries a scalar
  `intent`; nothing anywhere carries a list. The two shapes available are to make
  the spec's back-link the single source of truth and derive an intent's spec set
  from the store, or to let the intent carry the set explicitly. This record does
  not pick between them, and it does bind the requirement either must meet: the
  question "does this intent have an open spec" is answerable from committed
  records alone, and the link stays checkable in both directions rather than
  becoming one-sided. Whichever shape is chosen, the bidirectional-agreement
  check that `Reconcile` performs today has to keep meaning something.
- **Three readers assume one spec per intent and each has to learn otherwise.**
  `lint.SpecLinkIndex` (`internal/core/lint/speclinks.go`) maps each intent to one
  raw `spec_id` and resolves one id to one bucket; the `spec_lifecycle` rule
  reads that index; and the release cut's `staleIntents`
  (`internal/core/release/emit.go`) refuses a cut for any intent in `planned/`
  whose linked spec has closed. That last one is the sharpest: under this rule a
  planned intent with one closed spec and one open spec is the correct steady
  state, so the check as written would refuse every release taken during a
  partial delivery. It must ask whether the intent has any open spec, not whether
  its spec has closed. Left unchanged, this decision turns a lifecycle rule into
  a release wall.
- **A half-delivered capability is not announced, and the release notes lag the
  merge.** The changelog is composed from terminal folders, so an intent that
  stays planned while its second spec runs contributes no line, even though its
  first spec's code is on the default branch. That is the intended behaviour —
  the announcement should describe a capability, not a fraction of one — and it
  is a real cost: for one or more cycles, shipped code is live and unannounced.
  The honest record of it is the open spec, which is visible in the ledger the
  whole time.
- **The fidelity audit runs once, over more than one spec's delivery.** The
  intent-auditor compares a shipping intent's acceptance criteria against the
  delivered diff and runs at the ship transition. Under this rule that transition
  arrives later and the delivery it has to read spans every spec the intent
  owned. The audit's scope widens accordingly; what it is asked — did the
  delivery honour the criteria — does not change, and asking it once about the
  whole intent is closer to its purpose than asking it about a fraction would
  have been.
- **A bundle is the opposite cardinality and is untouched.** `kind:
  bundle-member` with a `bundle:` link is N:1 — several intents sharing one spec,
  moving together when that spec closes, governed by the invariant that every
  member belongs to one phase
  ([`04-surfaces/05-intent.md`](../../brief/04-surfaces/05-intent.md)). This
  record is 1:n — one intent, several specs. They are different relations, not
  two names for one thing, and nothing here changes the bundle rule or its
  invariant. Composing the two would be N:M; no case requires it, and this record
  does not authorise it.
- **An intent's specs may be scheduled in different phases.** The reason a second
  spec exists is that the work did not fit the cycle that carried the first, so
  requiring both in one phase would forbid the case the rule is for. That is a
  deliberate asymmetry with the bundle invariant above, and it follows from the
  cardinality rather than contradicting it: a bundle's members share one spec, and
  one spec has one phase.
- **The rule makes the honest path available; it does not make the dishonest path
  impossible.** Nothing forces the remainder spec to be minted. A session that
  closes the last open spec while criteria remain unmet still ships an intent that
  is not delivered, exactly as today. What changes is that it no longer has to:
  the state it was in is now representable, so the shortcut is a choice rather
  than the only move available. Mechanical detection of unmet criteria at close
  remains the fidelity audit's business and is not created here.
- **The two staged config keys stay staged and stay unread.**
  `intent.auto_link` and `intent.auto_ship` are documented as staged switches over
  behaviour that ships unconditionally
  ([`05-internals/03-configuration.md`](../../brief/05-internals/03-configuration.md)).
  This record changes *when* the ship happens, not whether it is switchable: the
  ship stays automatic and stays underived from configuration.
- **The finding stays open until the code follows.** iss-2609100508566552 records
  the defect and this record rules it; the verb, the schema carrier and the three
  readers above are the build, and the issue is resolved by the change that lands
  them, not by this one. Until then `spec close` still ships unconditionally, and
  a session meeting the half-delivered case should do what the field session did:
  stop and say so.

## Status note

**Accepted by the product thinker's ruling of 2026-09-15**, taken on
iss-2609100508566552 read together with its adjacent finding
iss-2609091732329046: the relation between an intent and its specs is 1:n; an
intent that has been thought through stands as written even when one spec cannot
deliver all of it; a spec that ships part of an intent is closed and a new spec
is minted for the remainder and attached to the intent; and an intent ships only
once all its specs are done. The routing — a decision record plus one brief
invariant, rather than an intent — was confirmed in the same ruling, on the
ground that this changes what closing a spec means everywhere rather than adding
a capability. The invariant is recorded in
[`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md).
