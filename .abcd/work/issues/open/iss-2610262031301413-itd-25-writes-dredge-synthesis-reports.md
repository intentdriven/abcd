---
schema_version: 1
id: "iss-2610262031301413"
slug: "itd-25-writes-dredge-synthesis-reports"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 139 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35"
remedy: "In itd-25 replace `.abcd/logbook/dredge/<timestamp>/` (scope, boundary with itd-36 and acceptance criteria) with `.abcd/.work.local/logs/dredge/<ts>/`, the location 07-memory.md and 04-universal-patterns.md give dredge's per-run report."
related_intents: [itd-25]
finding_ref: cfk-3335d3ec7aaafa4c
---

itd-25 writes dredge synthesis reports under .abcd/logbook/dredge/, the retired logbook location

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:35` — "writes `synthesis-report.{json,md}` to `.abcd/logbook/dredge/<timestamp>/`"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run logs, state and reports. It is retired and must not be re-minted"

itd-25's dredge synth writes synthesis-report.{json,md} to .abcd/logbook/dredge/< timestamp>/, and dredge list reads them back from there. The naming chapter says .abcd/logbook/ is retired and must not be re-minted. The draft's storage location cannot be built as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 139.
