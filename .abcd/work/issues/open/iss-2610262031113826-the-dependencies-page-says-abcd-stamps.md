---
schema_version: 1
id: "iss-2610262031113826"
slug: "the-dependencies-page-says-abcd-stamps"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 98 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:95"
remedy: "none (filed automatically)"
finding_ref: cfk-47448b6560f4ad1a
---

The dependencies page says abcd stamps `schema_version: 1` everywhere, but the configuration chapter says six of the fourteen committed records carry no stamp

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:95` — "abcd stamps `schema_version: 1` everywhere"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:431` — "eight of the fourteen committed records carry it, and six do not"

The dependencies page puts cross-version migration out of scope because abcd stamps `schema_version: 1` everywhere. The configuration chapter says the stamp is a convention, not an enforced rule: eight of the fourteen committed records carry it and six do not, the record-lint and docs-lint configuration among them. It notes that a migration would have no version to read on those six. 'Everywhere' and 'six do not' cannot both hold, and the migration plan rests on the stamp.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 98.
