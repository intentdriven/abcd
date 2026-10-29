---
schema_version: 1
id: "iss-2610262032491690"
slug: "07-memory-names-the-lifeboat-the-surface"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 321 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:81"
remedy: "In 05-internals/07-memory.md (section 4, near line 118) say the lifeboat is designed to publish curated project memory and that the gate's consumer is still future: the shipped pack's output shape carries no memory pages (07-library.md already says 'designed to', payload deferred)."
finding_ref: cfk-d9baf26f1d1778d7
---

07-memory names the lifeboat the surface that publishes curated project memory; disembark's pack inventory walks record families and brief sections and writes no memory

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:81` — "walk the source and its record families → the planned lifeboat file set"
- End B: `.abcd/development/brief/05-internals/07-memory.md:118` — "its real consumer is the lifeboat, the surface that publishes curated project memory/provenance (adr-35)"

07-memory routes the restrictive-licence gate to the lifeboat because it is the surface that publishes curated project memory and provenance. 02-disembark's pack inventory walks the source's record families and grounds brief sections into a planned file set whose output shape carries no memory store, and the pack runs only a destination gate and a secret scan before it writes. The lifeboat as packed publishes no curated memory for that gate to judge.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 321.
