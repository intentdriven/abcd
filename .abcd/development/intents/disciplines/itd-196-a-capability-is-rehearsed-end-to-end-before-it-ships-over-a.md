---
id: itd-196
slug: a-capability-is-rehearsed-end-to-end-before-it-ships-over-a
spec_id: null
kind: discipline
kind_notes: "A rule every delivery inherits: the last act before a ship is running the delivered thing the way its user will, over a real tree, in a snapshot that is thrown away."
suggested_kind: discipline
reclassification_history: []
builds_on: [itd-193]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A capability is rehearsed end to end before it ships, over a real tree, in a throwaway snapshot

## The rule

A capability is rehearsed by running it the way its user will: end to end, over
a real repository state, in a snapshot that is discarded afterwards. What the
run showed is recorded, whether or not it is good news.

A rehearsal is not a test. Tests answer whether each part behaves as its author
intended. The rehearsal answers a question no test is in a position to ask:
**is the thing usable for the purpose it was built for?**

It happens at three rungs, and they catch different failures.

1. **Per intent.** When an intent ships, run what THAT intent delivers. Catches
   a capability that does not work.
2. **Per intent, cumulatively.** Run what the PHASE can deliver so far, end to
   end, and rerun it as each intent lands. Catches pieces that work and do not
   compose. This is the rung that pays, and the one most easily skipped, because
   every part has just been proved individually.
3. **At the phase's close.** One full run of everything the phase set out to
   deliver, against the corpus it is meant to serve. Catches a whole that is
   correct and does not serve its purpose.

The cumulative rung is what turns a phase into something more than a list of
merged intents: at any point it can be RUN, so the phase carries a growing,
executable demonstration of itself rather than a claim about what it will do
once assembled. That demonstration is a deliverable, not a by-product.

Where a step cannot be automated, because it needs a human or a model, the
hand-off is written down and then performed. A step that cannot be scripted is
the most important one to make visible, never a reason to skip the run.

## Why

Measured, not reasoned. The cold-reading workstream shipped thirteen intents
across two phases. It ran six adversarial delta reviews, five fidelity audits,
and three review rounds on a single eval. Every property it claims was proved,
by mutation, against fixtures. Nothing was taken on trust.

The first end-to-end run took about fifteen minutes and found two things:

- the assembled input a reading is handed is about 9.8 MB, roughly 2.45 million
  tokens, so no reading can be given one (`iss-2608311501186646`);
- three of the four reading positions receive a byte-identical item set, though
  their definitions state four distinct objects, and one of those objects is not
  a subset of the repository at all (`iss-2608311501240566`).

Neither is visible from any document, any test, or any review. The evals run
over a miniature fixture repository of about thirty files, which is the right
corpus for asserting a firewall and the wrong one for asserting that the
artefact fits anywhere. The instrument could be, and was, entirely correct and
undeliverable at the same time.

The escalation is the whole point, and each rung found what the rung below could
not reach:

- a review asks whether the code is right;
- an audit asks whether the promise was kept;
- a rehearsal asks whether the thing works.

On this workstream the audits out-found the reviews, naming defects no builder
or reviewer had. The rehearsal then out-found the audits, in a fraction of the
time and cost, because the question it asks cannot be answered by reading
anything.

**And a per-intent rehearsal alone would have missed it.** The size defect
belongs to the assembler, shipped by an intent in an EARLIER phase, which passed
its own tests, its own review and its own audit. The three intents rehearsed
here deliver an ingest verb and two evals; none of them, run on its own, hands a
reading anything. Only the cumulative question, "what can this phase deliver
right now, end to end", reaches an artefact whose size nobody had cause to
measure. A defect that lives in the composition is invisible to every gate
scoped to a part, and the cumulative rung is the only one that looks at the
composition while there is still a phase left to change it.

## Why a throwaway snapshot

A real run writes real records. A rehearsal must not enter the record it
rehearses over, or the first exercise of a capability silently becomes a
production use of it, and the tree carries an artefact nobody commissioned.

This is [itd-193](itd-193-a-verifier-works-on-a-copy-no-agent-mutates-a-live-worktree.md)
at the scale of a whole capability rather than a single verifier: the copy is
the unit of safety, and here the copy is a snapshot of the tree at a real commit
with its own initialised repository, so that commit references, hashes and any
state keyed on a root commit resolve exactly as they would in earnest.

## The gate

- **Given** an intent whose ship ceremony is beginning, **when** the ceremony
  runs, **then** what that intent delivers has been exercised end to end over a
  real repository state in a throwaway snapshot, and what the run showed is
  recorded.
- **Given** an intent that has just landed in a phase, **when** its ship
  ceremony runs, **then** the phase's cumulative rehearsal is rerun over
  everything the phase can now deliver, and any behaviour that changed since the
  previous intent is recorded, including the case where the phase still cannot
  deliver anything end to end.
- **Given** a phase reaching its milestone, **when** it closes, **then** one
  full rehearsal is run against the corpus the phase is meant to serve, not a
  fixture, and its result is part of what the phase reports.
- **Given** a rehearsal step that needs a human or a model rather than a binary,
  **when** the rehearsal is written, **then** that hand-off is stated explicitly
  and performed, rather than the run being abandoned as unautomatable.
- **Given** a rehearsal showing the delivered thing cannot serve its purpose,
  **when** the result is recorded, **then** it is a finding of that intent or
  that phase, captured before the ship rather than discovered after it. The
  rehearsal proposes and the maintainer disposes, per
  [verifier-selects-gates-decide](../../principles/verifier-selects-gates-decide.md):
  a bad result does not by itself stop a ship, but shipping in ignorance of it
  is what this rule ends.

## Fit

[itd-195](itd-195-a-claim-about-how-the-code-behaves-is-executable-or-it-is-no.md)
is the complement rather than the parent. It makes a claim about a COMPONENT
executable. This rule makes the claim that the ASSEMBLED WHOLE works executable,
which no per-component claim implies: every part of the cold-reading instrument
was individually established and the composition was unusable.

[loud-staging](../../principles/loud-staging.md) is why the result is recorded
either way. A rehearsal whose bad news goes unrecorded is a stage that no-ops
quietly, and the temptation to leave it unrecorded is strongest exactly when it
matters most.

[script-first-mvp](../../principles/script-first-mvp.md) is why the rehearsal is
a script and not a verb. The flow has four steps and only three can be taken by
a binary; writing it down is what makes the hand-off visible, and a tool rung is
earned later against a corpus of runs rather than designed ahead of one.

## Staging

**Adopted 2026-08-31 at the documented-protocol rung.** The rule binds new work
immediately. There is no detector and no verb: the protocol is the gate, and a
rehearsal is a hand-run whose script belongs in the local tier until enough of
them exist to say what a general one would look like.

One rehearsal exists, of the cold-reading instrument, and it is a phase-close
run rather than the ladder this record describes: the per-intent and cumulative
rungs were never taken, which is why a defect from an earlier phase surfaced at
the end of a later one. Its script is not promoted out of the local tier. Promotion is a later rung and is not assumed
here: what a rehearsal looks like for a capability with no model in the loop, or
for one whose real corpus cannot be snapshotted cheaply, is unknown and will be
learnt from the second and third runs rather than guessed at now.

The existing corpus does not conform. No shipped intent in this repository has
been rehearsed, and none is assumed to have been.
