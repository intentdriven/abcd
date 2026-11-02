---
id: itd-87
slug: recurrence-escalation-in-capture
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
severity: minor
---

# A Finding That Comes Back After You Closed It Tells You Something, And abcd Stops Throwing It Away

## Press Release

> **abcd's capture ledger learns the difference between a duplicate and a recurrence.** When a finding arrives that matches one already dispositioned — resolved or wontfixed — the ledger no longer quietly folds it into the closed entry. It escalates: a new entry, citing the closed one, saying plainly that the condition which produced the original finding is still present. A closure carries an implicit claim that the thing was dealt with, and a recurrence is the cheapest available evidence that the claim is false. Today that evidence is exactly what deduplication destroys.

> "I closed an issue as fixed, and three weeks later the same thing surfaced again," said Bob, who works the ledger. "The tool told me it was a duplicate of the one I'd already closed — which is true, and completely the wrong thing to say. It wasn't a duplicate. It was the fix not holding. I want the ledger to tell me my closure was wrong, not to reassure me that it already knows."

## Why This Matters

A ledger's instinct is to dedupe, because a ledger's usual enemy is noise. But dedupe applied to a *dispositioned* entry is not noise control — it is evidence destruction. It converts "we were wrong to close this" into "we already know about this", and the second sentence is a lie the first would have caught. The failure mode is a ledger that grows quieter as it grows less accurate: the more confidently things are closed, the less able the ledger is to report that closing them was a mistake.

This is the one element of the ABCD sensemaking method with no counterpart anywhere in abcd's record (see [the method note](../../research/notes/2026-07-13-abcd-sensemaking-method.md)). The method states it as a prohibition on the reader — be deliberately amnesiac, re-raise tensions you have surfaced before, because one that keeps re-surfacing after it was supposedly resolved is the signal that the resolution was false. The principle [`recurrence-is-signal`](../../principles/recurrence-is-signal.md) states it as a rule. This intent is the mechanism: the principle's promotion from convention to tool.

It also unblocks itd-86. A blind cold reading re-raises old tensions *by design*; pointed at a ledger that dedupes them, it produces a detector fighting its own store. The escalation behaviour is what makes the re-raising useful rather than noisy, which is why the two are recorded together.

## What's In Scope

- Matching an incoming capture against **dispositioned** entries (`resolve`, `wontfix`), not merely against open ones.
- On a match, escalating rather than deduping: a new entry that cites the closed one and states that the condition persists.
- Distinguishing the two recurrence kinds, because they are different facts: a recurrence after `resolve` says the fix did not hold or was never made; a recurrence after `wontfix` says the accepted cost is still being paid.
- Keeping the human as the gate: an escalation is a claim that the closure is in question, never an automatic reopening. "Still wontfix, and here is why" must remain available — and is a stronger record than the original wontfix, having been made against evidence of persistence.

## What's Out of Scope

- **Automatic reopening or reversal of a disposition.** Per [`verifier-selects-gates-decide`](../../principles/verifier-selects-gates-decide.md), the escalation is a proposal.
- **Deduplication of *open* entries.** Unchanged; two live reports of the same open thing are a duplicate and should be treated as one.
- **Making the matcher clever.** A fuzzy matcher that invents recurrences is worse than no matcher, since it manufactures false evidence against closures the human made correctly. The matching rule should be conservative and explicable, and its false-recurrence rate is the thing to watch.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a capture matching an entry previously dispositioned `resolve`, **when** it is filed, **then** a new entry is created citing the resolved one and marked as a recurrence — and the capture is **not** folded into the closed entry.
- **Given** a capture matching an entry previously dispositioned `wontfix`, **when** it is filed, **then** it is escalated as a recurrence distinguishable from the `resolve` case, since the two carry different meanings.
- **Given** a capture matching an entry that is still **open**, **when** it is filed, **then** it is deduplicated as today — the new behaviour applies only to dispositioned entries.
- **Given** an escalated recurrence, **when** the human dispositions it `wontfix` again, **then** that is accepted and recorded against the evidence of persistence — the ledger never reopens or reverses a disposition on its own.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [memory-knowledge SOTA note](../../research/notes/2026-10-23-memory-knowledge-sota.md).

- **Sentry ships the two recurrence kinds.** "A plain Resolve treats any later event as a regression" and the issue's status "will automatically change to `Regressed`"; archiving is "until escalating" by default, and an issue archived forever "will never be labeled as escalating" (Sentry docs). Resolved means re-raise on any recurrence; archived means re-raise on a spike, or never, by explicit choice.
- **GitHub code scanning takes the opposite view of a dismissal.** After a dismissal "the same code won't generate an alert" on the next run, in all branches, with reopening by hand only (GitHub docs). Contested between the two vendors; the draft's second criterion takes the Sentry side (a wontfix match is escalated), and Sentry's "archive forever" is the explicit opt-out the ledger does not yet have.
- **Reopen reasons split along the same seam.** The ICSE 2012 study groups reopens as resolved-as-not-fixed (BY DESIGN, NOT REPRODUCIBLE, WON'T FIX) reopened on new information, resolved-as-fixed ("mostly regression bugs"), and process reopens; bugs from code-analysis tools were reopened 0.52 to 0.73 times the baseline, partly because such tools "do not argue that a bug should be fixed, once it has been closed as WON'T FIX" (Zimmermann et al.). A detector that cannot see the closure keeps arguing, which is what this intent asks of it.
- **The matching key is a declared fingerprint.** Sentry's fingerprint rules are `matcher:expression -> values`, "the first matching rule in the list is applied", and grouping on a changing message "can produce really bad groups"; SARIF versions its `partialFingerprints` and names baseline states `new`, `unchanged`, `updated`, `absent` (the verification SOTA note).
- **Locally the mechanism is three-quarters built.** The filing-time match already compares a capture against "every open and resolved issue", writes a `duplicates:` link and never refuses; wontfix records are excluded as "work nobody is doing" (`internal/core/capture/match.go`). A hit on a resolved record is today labelled a duplicate, which is the sentence this intent calls wrong.
- **Path 2**: on the shipped match, a `recurs:` relation (with its kind) in place of `duplicates:` when the candidate is resolved, wontfix added to the candidate set as the second kind, surfaced at capture time and counted in `abcd audit`, with Sentry's state semantics as the named model and, if the product thinker adds it (open question below), an explicit per-record silence as the only suppression, an exception to criterion 2; the matching rule stays the ledger's 0.6 term-overlap, versioned as SARIF versions keys. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

- **What counts as "matching"?** The conservative rule (exact same detector plus same location) under-reports; anything looser risks false recurrences. This is the crux of the build and should be settled with a corpus of real closures before implementation, per the acceptance-corpus discipline.
- Whether an escalation should surface at capture time, at `abcd audit` time, or both.
- (SOTA 2026-10-23) Should a `wontfix` be able to declare itself silent, as Sentry's "archive forever" does, so an accepted cost does not escalate on every restatement, with "escalate" the default and the silence written by a person with a reason? And should a wontfix recurrence escalate on the first match or, as Sentry's "until escalating" does, on a count?
