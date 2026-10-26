---
schema_version: 1
id: "iss-2610262038256056"
slug: "the-voyage-entry-has-the-voyage-record"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 793 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-19a7823fd42630fc
---

The voyage entry has the voyage record every disembark and embark run; the configuration chapter says only disembark runs are logged and the embark side is staged.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:4` — "an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run)"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:500` — "every disembark run; nothing else is written anywhere under the store, and an"

The voyage glossary entry defines the voyage as an append-only record of every disembark and embark run, and lays out `embark/provenance.json` and `embark/from/<timestamp>/`. 03-configuration says what ships is an append-only manifest log of every disembark run, that nothing else is written under the store, and that an embark-side record is staged. Embark runs are either recorded there or not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 793.
