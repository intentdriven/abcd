---
schema_version: 1
id: "iss-2610262037066238"
slug: "the-brief-index-says-spc-references"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 697 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:7"
remedy: "none (filed automatically)"
finding_ref: cfk-526d8c75e53c26e0
---

The brief index says spc references trace the predecessor's spec store and are not delivery claims; the build chapter cites a spec of this repository for what has landed.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:7` — "`spc-N` references throughout the brief trace"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:860` — "the design record: spc-2609202134338445, whose Progress section says which pieces have landed."

README.md says `spc-N` references throughout the brief trace the predecessor implementation's spec store, as design input and not as delivery claims. 34-build.md cites spc-2609202134338445 as its design record, whose Progress section says which pieces have landed: a spec in this repository's own store, used as a delivery reference. The README's blanket rule misdescribes the spec ids this repository mints.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 697.
