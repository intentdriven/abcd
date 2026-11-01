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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).
The ground was largely covered by the
[2026-08-19 forge-sync survey](../../research/notes/2026-08-19-issue-ledger-forge-sync-sota.md)
and the [mirror pilot](../../research/notes/2026-08-19-forge-mirror-pilot.md);
only the pull direction's mechanics were researched.

- **Reused**: import is a proposal a human accepts by an explicit act that
  writes the canonical record with provenance (forge id, actor, timestamp);
  ungated autonomous filing is the documented failure; a content-derived
  fingerprint makes a repeat run a no-op; the pilot found an unknown
  `forge_id:` field makes a record invisible to its own tooling, so the field
  must be schema'd first.
- **The forge's issue list contains pull requests**: "GitHub's REST API
  considers every pull request an issue"; drop them by the `pull_request`
  key, and note a pull request's issue id is not its pull-request id. Filters
  available: `state`, `labels`, `since`, `creator`, `assignee`, `type`;
  `per_page` up to 100 (issues REST).
- **Budget**: 5,000 requests an hour for a signed-in person, 60
  unauthenticated, 1,000 per repository for `GITHUB_TOKEN` in Actions (rate
  limits page).
- **No forge-native "two lenses confirmed" signal exists**; a label is the
  only durable filterable marker a hunt can leave, and anyone with triage can
  set it (inference from the filters above). The survey's negative evidence
  informs the autonomous-adoption question without deciding it: every
  production precedent found gates on a human, and none adopts autonomously;
  the call stays the interview's.

**Path 2**, unchanged from the standing decision: a native `capture adopt`
over the existing mint, shaped to the import conventions already chosen, with
the forge read as one adapter call. The independent fit-challenge runs at
plan time.

## Open Questions

- (SOTA 2026-10-23) Which marker a hunt leaves that adoption filters on, given
  a label is writable by anyone with triage, and whether adoption records who
  set it.
- Surface spelling: `capture --github-issues` vs a `capture adopt` subcommand —
  capture's bare-text-mints footgun makes namespace additions delicate.
- May dual-validated bughunt issues (two adversarial lenses already confirmed)
  adopt autonomously, or is every adoption human-gated per
  `verifier-selects-gates-decide`?
- Does adoption annotate/close the GitHub issue with the minted `iss-N`
  backlink, or leave the GitHub side untouched?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
