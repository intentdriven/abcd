---
schema_version: 1
id: "iss-2610262033366548"
slug: "the-capture-chapter-edits-ledger-records"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 423 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:160"
remedy: "none (filed automatically)"
finding_ref: cfk-f8d61c3fc2aff4a3
---

The capture chapter edits ledger records after they are filed, while universal patterns classes ledger entries as append-only and unchangeable once created.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:160` — "a resolved record's edges are history and stay editable"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:195` — "capture issue-ledger entries (immutable post-create)"

06-capture has linking rewrite a record's blocked_by in place, in any status folder, because a resolved record's edges stay editable. Resolving, wontfix, the remedy verb and deferral rewrite records too. 04-universal-patterns puts capture issue-ledger entries in the append-only class, never changed after creation. A lifecycle rule built on that taxonomy would refuse edits the capture verbs make by design.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 423.
