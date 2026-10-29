---
schema_version: 1
id: "iss-2610262038344053"
slug: "the-glossary-retires-phase-for"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 804 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/phase.md:18"
remedy: "In 03-configuration.md replace 'comes in a later phase (itd-13)' and the other 'later phase' sequencing mentions (user-scope memory, schema migration) with 'deferred' or the intent that carries it."
finding_ref: cfk-c370fe4ab0eb5865
refines: [iss-2609251618079479]
---

The glossary retires "phase" for dependencies plus the lifecycle shelves; the configuration chapter still schedules work into "a later phase".

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/phase.md:18` — "the sequencing layer is dependencies plus the lifecycle shelves"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1215` — "Scheduled sync comes in a later phase (itd-13)."

phase.md is superseded: sequencing is now dependencies plus the lifecycle shelves, and unqualified "phase" carries only the retired sense. 03-configuration still sequences live work by it: scheduled sync comes in a later phase (itd-13), and so do schema migration and user-scope memory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 804.
