---
id: adr-51
slug: intents-declare-mechanism-and-scope
status: accepted
date: 2026-08-22
supersedes: null
superseded_by: null
related_intents: [itd-142]
related_rfcs: []
related_adrs: [adr-2, adr-30]
---

# ADR-51: An intent can declare its mechanism claim and its scope conditions

## Context

An intent's press release commits to an outcome, and its acceptance criteria
commit to a verifiable bar — but nothing in the shape asks *why the authors
expect the mechanism to work*, or *under what conditions the claim holds*.
The gap shows up downstream: when a shipped intent underdelivers, the audit
can compare promise to delivery but not expectation to mechanism, because the
expectation was never written; and when a capability is reused outside the
conditions its design assumed, nothing recorded says the assumption existed.
The brief-creation interview design surfaced both: an elicited claim arrives
with grounds ("we expect this to work because…") and with boundaries ("this
holds while…"), and the intent shape had nowhere to put either.

## Decision

**The intent shape gains two optional sections**, documented in the
[`/abcd:intent` surface page](../../brief/04-surfaces/05-intent.md):

1. **`## Mechanism`** — the mechanism claim: why the authors expect this to
   work, stated as a falsifiable expectation ("we expect X because Y"), not a
   restatement of the outcome.
2. **`## Scope Conditions`** — the conditions under which the intent's claim
   holds: the population, platform, scale, or assumptions the design leans
   on, so a reuse outside them is a visible re-decision rather than a silent
   stretch.

Both sections are **optional and unenforced**. Whether either is ever
required — for every intent, for `severity: major` intents, at the planning
gate — is a discipline question, explicitly deferred; if enforcement comes,
it arrives as its own record on the itd-84/itd-1 pattern (a rule with a
staged gate), not as a side effect of this ADR.

## Alternatives Considered

1. **Fold mechanism into "Why This Matters".** Rejected: that section argues
   value, not mechanism; blending them lets an intent argue importance while
   staying silent on why the approach should work.
2. **Require the sections immediately.** Rejected: a required section with no
   calibrated reviewer becomes boilerplate; itd-84's promotion ladder shows
   the order that works — shape first, discipline after evidence.
3. **A separate record family for mechanism claims.** Rejected: the claim is
   the intent's own, and adr-30's information architecture routes a
   feature-scoped statement to the feature's record, not a parallel store.

## Consequences

- The surface page's template gains both sections with their one-line
  contracts; existing intents are untouched (optional means absent is valid).
- The intent audit (promise vs delivered) gains material when the sections
  are present: a delivered-but-mechanism-refuted outcome becomes expressible.
- The interview (itd-142) gets a committed destination for elicited grounds
  and boundaries at intent capture.
- Enforcement, if it ever comes, is a later, separate decision with its own
  evidence.
