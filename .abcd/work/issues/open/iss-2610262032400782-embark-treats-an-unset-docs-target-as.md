---
schema_version: 1
id: "iss-2610262032400782"
slug: "embark-treats-an-unset-docs-target-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 301 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:131"
remedy: "none (filed automatically)"
finding_ref: cfk-808bde266ee53627
---

embark treats an unset docs.target as agents_md; configuration says skip is the default

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:131` — "`agents_md` or no choice saved gives `AGENTS.md`"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:66` — ""skip" is the default, so a default install names abcd"

The embark chapter says that, read with setup's reader, a target with no docs.target saved gets the marker block in AGENTS.md. The configuration chapter says skip is the default, so a default install names abcd in no conventions file. The same reader cannot give agents_md for an unset key in one place and skip in the other. Embark would plant abcd's block in a repository that never chose one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 301.
