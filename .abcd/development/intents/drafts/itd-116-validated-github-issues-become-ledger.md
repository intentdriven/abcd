---
id: itd-116
slug: validated-github-issues-become-ledger
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: minor
impact: additive
---

# Validated GitHub Issues Become Ledger Entries Without Retyping

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

An autonomous bug-hunt files validated defects as GitHub issues by mandate, but
every workflow that acts on findings — plans, detectors, `drain` — reads only
the ledger. Until a human retypes each finding, it is invisible to the
machinery that fixes things. This intent makes adoption a tool: a capture
extension (`--github-issues` / `capture adopt`) selects externally filed
issues, reviews them host-delegated, and mints each adopted one into the
ledger via the existing capture path with provenance back to the GitHub
issue. The pipeline composes as stages: the hunt files → adoption mints →
drain (itd-82) triages what is now in the ledger, without drain ever knowing
GitHub exists.

Decomposition (itd-84, hand-run 2026-08-16): the trust rule (host-delegated
select/review, fail-closed binary ingest, core never touches the network) is
adr-25 plus the existing core boundary — a spec constraint, not a new ADR.
The stance (an agent proposes, the maintainer adopts) is the
`verifier-selects-gates-decide` principle, cited not re-filed. This intent
refines the recorded bughunt-hybrid decision ("adoption into the ledger is a
downstream human/fix step") into a capability. Recurrence/dedupe semantics
against already-captured `iss-N` will consume itd-87 when it lands.

**Considered and set aside:** folding this into `drain --github-issues`
(itd-82). Rejected because drain is an unbuilt draft already carrying a
stubbed dependency (itd-46), and the two flows point opposite directions with
different blast radii — drain consumes the ledger and opens PRs; adoption
feeds the ledger and mints entries. Whatever the final flag spelling, the
mint itself goes through the existing capture path — this surface must never
become a second way to create `iss-N` (one-canonical-primitive).

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Surface spelling: `capture --github-issues` vs a `capture adopt` subcommand —
  capture's bare-text-mints footgun makes namespace additions delicate.
- May dual-validated bughunt issues (two adversarial lenses already confirmed)
  adopt autonomously, or is every adoption human-gated per
  `verifier-selects-gates-decide`?
- Does adoption annotate/close the GitHub issue with the minted `iss-N`
  backlink, or leave the GitHub side untouched?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
