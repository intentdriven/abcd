---
id: adr-50
slug: framing-traces-never-enter-the-record
status: accepted
date: 2026-08-22
supersedes: null
superseded_by: null
related_intents: [itd-142, itd-143]
related_rfcs: []
related_adrs: [adr-36, adr-41, adr-55]
---

# ADR-50: Framing traces never enter the record, and automated reviewers never read them

## Context

The brief-creation interview (itd-142) elicits more than it commits: at a
conjectural question it widens the space to several candidate construals, and
the human declines most of them; along the way the exchange accumulates
framing traces — the phrasings tried, the construals rejected, the paths not
taken. Where that residue lands is a trust question, not a storage question.
If declined alternatives enter the committed record, every future reader —
human or automated — re-litigates them; if automated reviewers can read the
traces, review pressure reaches back into the elicitation and shapes what a
human is willing to consider out loud. The interview's design settled this as
the two-output rule; the rule outlives the interview surface, so it is
recorded as its own decision.

## Decision

**Elicitation has exactly two outputs, and only one enters the record.**

1. **The brief and the ledger receive committed products only** — content the
   human confirmed, with its provenance. Nothing tentative, nothing declined.
2. **Declined alternatives and framing traces go to a local ledger side** —
   the `.abcd/.work.local/` tier, per-worktree and never committed. The exact
   store shape is the itd-142 spec's to fix; the tier is not.
3. **Automated reviewers never read the local ledger side.** No review agent,
   validator, or lint consumes framing traces — not as context, not as
   evidence, not as calibration data. A human may read their own traces;
   machinery may not.
4. **The transcript is discarded** once the committed products and the local
   ledger side are written. There is no third output.

## Alternatives Considered

1. **Commit the traces as evidence** (a what-was-declined record beside the
   brief). Rejected: it converts every declined construal into a standing
   invitation to re-litigate, and it makes the elicitation performative — a
   human who knows every rejected phrasing is committed reviews their words
   instead of thinking.
2. **Let reviewers read traces under a flag** (opt-in review context).
   Rejected: the value of the rule is its unconditionality; a flag makes
   trace-visibility a negotiation, and the chilling effect returns the moment
   the flag exists.
3. **Discard everything uncommitted** (no local side at all). Rejected: the
   human's own traces have recall value to the human — the local tier
   preserves that without publishing anything.

## Consequences

- The brief stays a record of decisions, not deliberations — consistent with
  the brief's truth rule (everything in it is decided) and adr-5.
- Brief invariant 14 states the rule where invariants live; itd-142 consumes
  it, and its acceptance criteria assert the boundary behaviourally.
- The local ledger side joins the `.abcd/.work.local/` tier's existing
  contract: per-worktree, gitignored, merge-conflict-free.
- Any future reviewer or validator that wants elicitation context must be
  told no by design — a check that would benefit from reading traces is a
  check that must be redesigned to run on committed products.
