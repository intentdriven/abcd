---
schema_version: 1
id: "iss-2610262032428223"
slug: "universal-patterns-says-issue-ledger"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 305 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:195"
remedy: "none (filed automatically)"
finding_ref: cfk-1eefd1f7f776665d
---

universal patterns says issue-ledger entries cannot change after creation; launch has the deferral verb write a waiver into an open record

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:195` — "capture issue-ledger entries (immutable post-create)"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:598` — "`deferral_reason`, both schema-accepted keys, written by the ledger's deferral verb"

The artefact-lifecycle table puts issue-ledger entries in the append-only class: never modified after creation. Launch's findings gate depends on records being changed after capture: the ledger's deferral verb writes the deferred_after and deferral_reason pair into the record's frontmatter, and a record may be re-graded. A ledger whose entries never change could not hold either.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 305.
