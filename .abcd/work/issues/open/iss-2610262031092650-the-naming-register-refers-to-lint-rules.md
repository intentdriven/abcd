---
schema_version: 1
id: "iss-2610262031092650"
slug: "the-naming-register-refers-to-lint-rules"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 92 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:147"
remedy: "none (filed automatically)"
finding_ref: cfk-7f1147d3a9595a61
---

The naming register refers to lint rules by numbered codes (GL002, SD001, VR001), but the lint contract says rules are named, not numbered

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:147` — "and it is what the `GL002` forbidden-synonym rule reads"
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "rules are named, not numbered, and their definitions live in the two JSON configs above"

The naming register names a live rule by number, saying the glossary is what the `GL002` forbidden-synonym rule reads. It also reserves `SD001` and `VR001` for unbuilt checks. The lint contract says rules are named, not numbered, are defined only in the record-lint and docs-lint JSON configs, and that the record should not carry a hand-kept enumeration of rules. The two pages identify the same rules by different schemes, so a numbered code in the register maps to nothing in the engine's own vocabulary.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 92.
