---
schema_version: 1
id: "iss-2608230847432287"
slug: "the-adversarial-review-floor-assumes-below-the-line-artefact"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "record-review"
found_at: ".abcd/development/principles/adversarial-review-scales-with-blast-radius.md"
details: "adversarial-review-scales-with-blast-radius sets a floor: ledger captures, comments and routine pull requests are never blocked on adversarial review because they 'are covered by the gates they already have'. On 2026-08-22/23 four errors were made in exactly those below-the-line artefacts, and record-lint, docs-lint and lint-reviews passed on every one. The floor's empirical claim is false as written. The floor's PURPOSE stands: mandatory review on captures would suppress the ledger, exactly as the principle argues. What is wrong is the stated reason, not the placement."
suggested_fix: "Correct the floor's justification rather than moving the floor. Below-the-line artefacts are unblocked because review friction would cost more than the errors do, not because their gates cover them. Optionally add the narrow bound: a peer session already present is a near-zero-friction reviewer, so the cost argument does not apply to it. Weigh that bound against the thin evidence recorded in iss-2608230847432286 before adopting; a maintainer decides."
related_issues: ["iss-2608230847432286", "iss-2608230752354926"]
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: restate the floor reason in adversarial-review-scales-with-blast-radius; peer-review convention held (SPLIT, HOLD)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the adversarial-review floor assumes below-the-line artefacts are covered by their gates, and today they were not

[adversarial-review-scales-with-blast-radius](../../../development/principles/adversarial-review-scales-with-blast-radius.md)
draws a floor, and gives a reason for it:

> Artefacts below that line — ledger captures, comments, routine pull requests
> — are never blocked on adversarial review: they are covered by the gates they
> already have, and capture in particular must stay frictionless.

Two clauses, and they are doing different work. "Capture must stay
frictionless" is a cost argument and it is sound. "They are covered by the
gates they already have" is an empirical claim, and it is false.

## The counter-evidence

Four errors on 2026-08-22/23, every one in a below-the-line artefact, every one
passing `record-lint`, `docs-lint` and `lint-reviews`:

- A capture filed as a standalone process observation when it refines iss-213,
  which is what gives it its weight. Routing, not form.
- A proposed new principle that `one-canonical-primitive` forbids, because
  `enforcement-claims-are-facts` already owned the ground.
- Two ADR links in an issue body resolving to nothing, because `links_resolve`
  does not run over `.abcd/work/` (iss-2608230752354927).
- Record repairs and a capture written into a shared checkout another session
  was about to branch from (iss-2608230847432285).

The first two are the interesting pair. Both are errors of *routing and
relation* — which record this belongs to, which existing record already owns
it — and no deterministic gate can see them, because both artefacts were
perfectly well-formed. A linter checks whether a record is shaped correctly. It
cannot check whether it is the right record.

## What follows, and what does not

The floor should stay where it is. Its own argument for the placement is
correct and this changes none of it: mandatory review on captures would
suppress the ledger, recording a finding must cost less than fixing it, and
reviewers trained on mandatory verdicts over trivia skim the one that matters.

What should change is the stated reason. Below-the-line artefacts are unblocked
because the friction would cost more than the errors do — not because their
gates cover them. Keeping the false clause is itself an instance of the class
recorded in iss-2608230847432286: a reader who believes the gates cover these
artefacts stops compensating, which is the precise harm that record describes.

## The bound, offered weakly

A peer session already present and already communicating is a reviewer at
near-zero friction, so the cost argument the floor rests on does not apply to
it. That suggests a standing convention: announce what you are working on to
concurrent peers and engage with what they announce.

This is offered as a proposal, not a finding, and the evidence for it is thin.
iss-2608230847432286 records why: four concurrent sessions of one model in one
repository is not the independence
[evaluator-outside-the-loop](../../../development/principles/evaluator-outside-the-loop.md)
asks for, a shared blind spot would have been invisible to all four, two of the
catches were a peer naming a record the author had not read (recall, not
review), and at least one was luck. Peers also produced errors — a record-merge
proposal that did not survive scrutiny, a threshold that collapsed on a wider
sweep, a misidentified record owner.

A second reading is available and the record should not suppress it. Today's
coordination ran entirely on a convention with no mechanism behind it: four
sessions wrote in one repository, nobody lost work, and what held was "a diff
you did not make is a peer's work" — a sentence in AGENTS.md that nothing
enforces. That supports either conclusion. The convention is sufficient and
needs no machinery, or the convention got lucky four times and the sample is
too small to tell which. The isolation property the same document claims did
fail on the day (iss-2608230847432285), and the convention is what absorbed it,
so at minimum the two are not independently reliable.

Routing, per itd-84: the capability half is already owned by itd-33
(coordination contract) and iss-2608220750029993 (presence detection), so a
second intent beside itd-33 would be the third copy. The stance half is a
principle bound plus a line in AGENTS.md § Concurrent sessions, whose five
existing concurrent-session records are all hazard-framed and none of which
treats a peer as a resource. Verdict SPLIT, HOLD on the intent.

Promotion trigger, per [recurrence-is-signal](../../../development/principles/recurrence-is-signal.md):
if this recurs among sessions that are not coincidentally co-present, the
convention has earned more than a caveat.
