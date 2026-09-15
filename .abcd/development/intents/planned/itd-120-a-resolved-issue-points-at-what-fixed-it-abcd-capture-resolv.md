---
id: itd-120
spec_id: spc-25
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: major
impact: additive
slug: a-resolved-issue-points-at-what-fixed-it-abcd-capture-resolv
---

# A Resolved Issue Points At What Fixed It

## Press Release

A resolved issue now points at what fixed it. `abcd capture resolve <iss-N>
"<note>" --impact fix --intent itd-N --spec spc-N --commit <sha>` stamps the
structured provenance the schema has modelled all along — the `resolved_by`
pointer that was parsed on read and written by nothing. "When I close an issue
I name the intent, spec, or commit that fixed it, and six months later the
trail is still there," says Nia, facilitator.

## Why This Matters

Step 12 of the twelve-step record walk closes the issue but loses the trail:
`ResolvedBy{Intent, Spec, Commit}` is parsed on read while `Resolve` writes
only `resolution` and `impact`, so a resolved issue asserts it was fixed in
prose but cannot point at what fixed it — while the intent side of the same
record store binds its verdicts to a SHA-256 receipt. One record store, two
evidence standards. This intent closes the `resolved_by` half of iss-245 (the
`promoted_to` half is itd-119); when both ship, iss-245 itself resolves *with*
provenance — the first entry in the ledger to carry the trail.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** an open issue, **when** resolve runs with any of `--intent itd-N`,
  `--spec spc-N`, `--commit <sha>`, **then** `resolved_by` is written with
  exactly the supplied members, alongside `resolution` and `impact`, in the
  same atomic transition to `resolved/`.
- **Given** `--intent` or `--spec`, **then** the id must *exist* in its record
  store (any bucket, open or closed); `--commit` is shape-checked only (7–64
  hex characters). An unknown id or malformed value refuses the whole
  transition — nothing written, the issue stays open.
- **Given** no provenance flags, **then** resolve behaves exactly as today: no
  `resolved_by` key is written at all — provenance is optional, never
  defaulted, never guessed.
- **Given** `--json`, **then** the result reports the transition and the
  written `resolved_by` members.
- **Given** the change, **then** `wontfix` is untouched — a non-action ships
  nothing and points at nothing.
- **Given** the sweep, **then** `commands/capture.md` and the issues README
  document the provenance flags.

## SOTA

Close-with-reference is standard tracker practice (GitHub "fixes #N" commit
links, GitLab closing patterns, Jira issue links); nothing importable — the
provenance lands in abcd's native frontmatter schema, already modelled.
**Chosen path: bespoke**, extending the existing `Resolve` transition and its
atomic-write machinery. No new dependency.

## Open Questions

_None gating. Backfilling `resolved_by` onto already-resolved issues was
ruled out of scope at the grill, 2026-08-16 — the verb closes the gap going
forward._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
