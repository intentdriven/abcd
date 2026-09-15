---
id: itd-119
spec_id: spc-24
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: major
impact: additive
slug: an-issue-graduates-into-an-intent-without-retyping-abcd-capt
---

# An Issue Graduates Into An Intent Without Retyping

## Press Release

Capture stops being a dead end. When a one-line issue turns out to be a
capability, `abcd capture promote <iss-N>` graduates it into an intent draft
without retyping — the minted draft carries the issue's text, the issue is
stamped with the `itd-N` it became, and the trail survives in both directions.
"I used to hesitate at capture time — issue or intent? Now I capture everything
as one line and promote the ones that grow up," says Iris, product lead.

## Why This Matters

Step 2 of the twelve-step record walk — *decide it is a capability* — has no
verb. The schema already models the graduation: `promoted_to` is validated
against `^itd-[0-9]+$` and documented as "the itd-N this issue graduated into",
yet no verb writes it and no issue in the ledger carries it (refines the
`promoted_to` half of iss-245; the `resolved_by` half is a sibling intent). The
current `commands/capture.md` promote path is skill-orchestrated retyping with
no back-link. A native verb closes the forced intent-vs-issue choice the
"Which ledger?" note imposes at the moment of lowest information: capture now,
decide later, promote the ones that grow up.

The press release of the *promoted* intent is stated at its planning interview,
not at promotion time — promotion mints a draft with the standard placeholder,
so the promote moment stays one cheap command.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** an issue in the ledger, **when** `abcd capture promote <iss-N>`
  runs, **then** a new intent draft is filed under `drafts/` — its slug reused
  from the issue's slug, its body carrying the standard placeholder Press
  Release section plus a by-id pointer ("Graduated from iss-N") and the issue's
  one-line summary, never a copy of the issue body (SSOT) — and the issue's
  `promoted_to` is stamped with the minted `itd-N` in the same invocation.
- **Given** an issue in *any* status (`open/`, `resolved/`, `wontfix/`),
  **when** promote runs, **then** the graduation succeeds and the issue keeps
  its folder — promotion is orthogonal to fix-status and is not resolution.
- **Given** an issue already carrying `promoted_to`, **when** promote runs
  again, **then** the verb refuses and reports the existing `itd-N` — no
  duplicate drafts.
- **Given** a malformed or unknown `iss-N`, **when** promote runs, **then** it
  fails structurally with a diagnostic and nothing is written.
- **Given** the two-store write, **when** promote executes, **then** it mints
  the draft first and stamps the issue second; a failure after the mint reports
  the orphan draft's path and the remedy — `capture promote <iss-N> --intent
  <itd-N>`, a stamp-only mode that links an *existing* draft instead of
  minting (which also serves "I already filed the intent by hand; link them").
  Concurrent ledger transitions serialize under the existing ledger lock.
- **Given** the minted draft, **then** its frontmatter records the source issue
  (field named in the spec), so the edge is two-sided.
- **Given** `--json`, **then** the result reports the issue id, the minted (or
  linked) intent id, and both repo-relative paths.
- **Given** the sweep, **then** `commands/capture.md` documents the native verb
  (replacing the skill-orchestrated-retyping paragraph),
  `02-constraints/04-naming.md` drops the design-target marker, and the
  `04-surfaces` registry reflects `promote` as shipped — coordinating with the
  sub-verb-table intent if its rows land first.

## SOTA

Promote-with-link is a ubiquitous tracker pattern (Linear convert issue →
project, Jira "convert to `epic`", GitHub issue → discussion); nothing
importable —
abcd's record stores are native. **Chosen path: bespoke**, thin Go over
existing primitives, reusing the intent-create core function
(one-canonical-primitive) and the capture ledger's transition lock. No new
dependency.

## Open Questions

_None gating. Grill findings (two-store ordering, seed shape) were confirmed
and folded into the criteria, 2026-08-16._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
