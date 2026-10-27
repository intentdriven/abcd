---
schema_version: 1
id: "iss-2610262039494979"
slug: "04-universal-patterns-classes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 889 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:195"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-b7c9cceafff165be
---

04-universal-patterns classes issue-ledger entries as immutable once created; itd-4's `capture resolve` moves an entry and adds a resolution note.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:195` — "capture issue-ledger entries (immutable post-create)"
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:36` — "move to `resolved/` with optional resolution note"

The lifecycle taxonomy puts capture issue-ledger entries in the append-only class: never modified after creation, old entries kept verbatim. itd-4 defines `capture resolve` as moving the record to `resolved/` with an optional resolution note, and gives every record an `updated` field. An entry that changes folder and gains text after creation is not immutable. The same table also lists the ledger as compounding-curated.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 889.
