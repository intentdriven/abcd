---
schema_version: 1
id: "iss-2610262039442943"
slug: "03-configuration-says-the-shared-tier"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 882 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:805"
remedy: "none (filed automatically)"
related_intents: [itd-2609151138388536]
finding_ref: cfk-15b192a28d7a0ec9
---

03-configuration says the shared tier holds 'the append-only decision log'; itd-2609151138388536 replaced that log with one file per decision.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:805` — "current orientation, the append-only decision log, the external-contribution runbook"
- End B: `.abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:25` — "longer a line appended to the bottom of one shared file. It is a record with"

The three-tier table says `.abcd/work/` holds the append-only decision log. itd-2609151138388536 (shipped) retires `.abcd/work/DECISIONS.md` and says a decision is no longer a line appended to one shared file but its own record in `.abcd/work/decisions/`. A reader of the tier table would look for a single appended log that no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 882.
