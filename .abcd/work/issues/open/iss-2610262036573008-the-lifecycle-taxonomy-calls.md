---
schema_version: 1
id: "iss-2610262036573008"
slug: "the-lifecycle-taxonomy-calls"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 687 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:195"
remedy: "none (filed automatically)"
finding_ref: cfk-a1847199eaf5935b
---

The lifecycle taxonomy calls issue-ledger entries immutable after creation, while the drain's user-visible route writes `related_intents` into the issue

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:195` — "capture issue-ledger entries (immutable post-create)"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:579` — "the draft, and the issue's `related_intents` naming it; nothing else"

04-universal-patterns puts capture issue-ledger entries in the append-only class, never modified after creation. The drain's hand-back table says a user-visible hand-back writes the intent draft and adds `related_intents` to the issue, which edits an existing issue record. The drain chapter also re-admits a handed-back issue only once a person changes its record. If the taxonomy holds, the drain's routing breaks it by design.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 687.
