---
id: itd-129
slug: forge-mirror-as-an-opt-in-adapter-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Forge mirror as an opt-in adapter: the ledger mirrors out to forge issues one way, the forge id lives in the record schema, closures without import self-heal, and the whole surface is loudly absent without forge auth

## Press Release

Alice's team lives on GitHub; Bob has no forge access at all. Both see the
same issues, because there is only one issue store: the committed ledger.
Alice runs the mirror and every open record becomes a forge issue titled by
its handle, carrying a pointer back to the canonical file — and the forge id
is written into the record, so a re-run touches nothing it already mirrored.
When Carol triages on GitHub and closes a mirror without importing the
outcome, the next pass reopens it with an explanation: the ledger owns
existence, content, and resolution, and the forge owns nothing. When a record
resolves in the ledger, its mirror closes itself, citing the handle. Without
forge credentials the whole surface says exactly that and does nothing —
Bob's abcd is complete without it.

## Why This Matters

The 2026-08-19 decision (ledger-canonical, one-way mirror, human-gated
import — see `../../../work/DECISIONS.md` and the SOTA survey at
`../../research/notes/2026-08-19-issue-ledger-forge-sync-sota.md`) was
hand-run the same day on iss-285/286/287 and graded in
`../../research/notes/2026-08-19-forge-mirror-pilot.md`. The pilot's verdict:
the topology survives contact, mirror-out is fully mechanical, self-healing
needed no state beyond the pointer and the two visible states — and the one
real obstacle is that the issue schema rejects the write-back field, which
makes the record invisible to its own capture surface. This intent automates
what the pilot proved and schemas what it exposed.

## Acceptance Criteria

- Given an issue record carrying `forge_id`, When capture, record-lint, or
  any ledger reader parses it, Then the field is schema-valid and
  shape-checked (positive integer, unique across the ledger), and an unknown
  frontmatter property still refuses loudly — closing pilot finding F-A
  without loosening strict parsing.
- Given open records without a `forge_id`, When the mirror-out runs, Then
  each gains exactly one forge issue (handle-titled, canonical path and
  topology footer in the body) and its id written back; a second run is a
  no-op (idempotence keyed on `forge_id`).
- Given a mirror closed on the forge with no corresponding import, When the
  reconciliation pass runs, Then the mirror is reopened with a comment naming
  the still-open canonical record (pilot F-C, proven by hand on #373).
- Given a record moved to resolved/ or wontfix/ whose `forge_id` mirror is
  open, When the pass runs, Then the mirror is closed with a comment citing
  the handle (pilot F-D — the direction the pilot could not exercise; its
  first real run completes the pilot).
- Given more than a handful of mirrored records, When any pass runs, Then
  forge state is read in one batched listing, not one API call per record
  (pilot F-C cost note).
- Given no forge credentials or no configured adapter, When any mirror verb
  runs, Then it reports the surface as unavailable and writes nothing — a
  loud no-op, never a partial pass.

## Open Questions

- Which rung first: `scripts/forge-mirror.sh` (the earned next step per
  script-first) or straight to `abcd forge mirror` verbs? The pilot was
  mechanical enough that the script rung may be short-lived — but the schema
  change (F-A) is Go-side either way and could ship first on its own.
- Import (`abcd forge import <n>`, writing a canonical record from a
  forge-side report with provenance): same intent or its own? The pilot
  never exercised import; filing it here unexercised would repeat the
  mistake this intent exists to avoid.
- Does the mirror cover the whole open ledger or an opt-in subset (label,
  severity floor)? The pilot deliberately bounded to three records; a full
  first mirror-out is ~140 issue creations in one pass.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
