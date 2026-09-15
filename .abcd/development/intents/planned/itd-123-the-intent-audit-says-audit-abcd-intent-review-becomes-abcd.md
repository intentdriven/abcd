---
id: itd-123
spec_id: spc-28
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-intent-audit-says-audit-abcd-intent-review-becomes-abcd
---

# The Intent Audit Says Audit

## Press Release

The verb that answers the product thinker's question finally says so.
"'Review' told me someone judged the code. What this verb actually answers is
whether I got what I asked for — that's an audit, and now it's named as mine,"
says Iris, product lead. `abcd intent audit` emits the same per-criterion
verdicts it always did; what changed is that the name now tells you whose
question it answers.

## Why This Matters

`abcd intent review` emits family 2 (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` /
`INCONCLUSIVE`) — per adr-40 that comparison is an audit, and the brief's own
mental model already calls it "the intent audit". The name crossed with
itd-85's verb, so a maintainer cannot reason from verb to bucket. This is
adr-40's first named rename: clean break, no aliases, no deprecation shims —
abcd is pre-1.0.0, `--impact breaking` drives version derivation, users
re-download. About 37 files reference the current name. The
`intent-fidelity-reviewer` agent becomes `intent-auditor` (the intent-grain
auditor of the three audit grains), and the `intent_review` task-class token
becomes `intent_audit`.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `abcd intent audit [<itd-N>]` and
  `abcd intent audit ingest --verdict-json <file>` replace the `review`
  spellings, and the old sub-verb is gone — no alias, no shim;
  `abcd intent review` fails as an unknown sub-command.
- **Given** the sweep, **then** every live reference moves:
  `commands/intent.md`, brief prose, the agent file and its registered name
  (`intent-fidelity-reviewer` → `intent-auditor`), the `intent_review`
  task-class token → `intent_audit` with its `04-naming.md`
  reserved-vocabulary row updated in the same change, code, tests, and the
  verb's JSON field names.
- **Given** the sweep boundary, **then** historical and dated records are not
  rewritten — ADRs, dated research notes, resolved issues, shipped intents,
  and `DECISIONS.md` keep the old name as history; the brief (current-state
  per adr-5), commands, code, and `docs/` move.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  `intent` sub-verb table row flips in the same change and `record-lint`
  exit 0 proves the migration complete; landing this rename before that
  check is armed is forbidden.
- **Given** stored artefacts (receipts, previously ingested verdicts),
  **then** they are not rewritten, and any parser that reads them continues
  to accept them.
- **Given** the release record, **then** the CHANGELOG entry is breaking and
  version derivation reads it.

## SOTA

Breaking CLI renames without aliases are the pre-1.0.0 norm (semver §4;
established precedent in this repo via iss-171's `--impact breaking` path);
alias/deprecation machinery was considered and rejected in adr-40 (it
collides with the change-narration ban). **Chosen path: bespoke sweep**,
proved complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The agent name (`intent-auditor`) was ruled by the maintainer
at the walk, 2026-08-16._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
