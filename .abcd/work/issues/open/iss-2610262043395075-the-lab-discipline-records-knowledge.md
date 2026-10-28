---
schema_version: 1
id: "iss-2610262043395075"
slug: "the-lab-discipline-records-knowledge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1152 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-2609251624540864, itd-2609151138388536]
finding_ref: cfk-ab2e69d175a6b14b
---

The lab discipline records knowledge through 'a decisions-log line'; itd-2609151138388536 retired the append-only decision log

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-2609251624540864-a-lab-runs-one-procedure-from-intention.md:47` — "or decision record that cites the lab, a decisions-log line"
- End B: `.abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:25` — "longer a line appended to the bottom of one shared file"

The lab procedure's record stage admits knowledge through, among other ceremonies, a decisions-log line. The shipped decisions intent rules that a decision is no longer a line appended to one shared file: each is its own record and DECISIONS.md is retired outright. One of the lab's ceremony steps names a surface that no longer takes entries.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1152.
