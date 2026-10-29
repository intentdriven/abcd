---
schema_version: 1
id: "iss-2610262031598665"
slug: "02-dependencies-says-abcd-stamps-schema"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 205 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:69"
remedy: "In 02-dependencies.md Out of scope, replace 'abcd stamps schema_version: 1 everywhere' with what holds: each record family stamps its own current version (run state is at 14 and migrates older files on read), and cross-version migration of the record store stays itd-9; the same edit resolves iss-2610262030364518."
related_intents: [itd-166]
finding_ref: cfk-8b0815e4ace235d1
---

02-dependencies says abcd stamps schema_version: 1 everywhere; itd-166 moves run state to schema 14.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:69` — "Run state moves to schema 14."
- End B: `.abcd/development/brief/02-constraints/02-dependencies.md:95` — "abcd stamps `schema_version: 1`"

02-dependencies states that 'abcd stamps `schema_version: 1`' everywhere and leaves migrators to a draft intent. itd-166 records 'Run state moves to schema 14.': a version-13 file is read as not recorded and written back at 14. Both cannot describe the current record shapes; the brief's out-of-scope note is out of date against the versioned schemas the intents have shipped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 205.
