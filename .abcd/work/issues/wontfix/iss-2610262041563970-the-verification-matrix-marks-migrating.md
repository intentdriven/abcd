---
schema_version: 1
id: "iss-2610262041563970"
slug: "the-verification-matrix-marks-migrating"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1035 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:71"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-d6ed8bca7734e28b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later decision record already says which holds: no verb reads the flat issues file, and itd-4's own audit notes say so."
---

The verification matrix marks migrating a flat issue list into the ledger as staged with no verb; shipped itd-4 scopes that migration.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:71` — "promoting a pre-ledger flat issue list into the structured ledger with provenance naming the source file"
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:46` — "promotes existing `.abcd/.work.local/issues.md` entries to the structured ledger on first run after install"

The matrix's capture-migration row is staged. It says the binary registers no verb that reads a flat list, and that `capture migrate` only rewrites back-links. The shipped itd-4 includes a migration path in its scope: dev-sync promotes existing .abcd/.work.local/issues.md entries into the structured ledger on the first run after install. One document says this behaviour shipped and the other says it does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1035.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later decision record already says which holds: no verb reads the flat issues file, and itd-4's own audit notes say so.
