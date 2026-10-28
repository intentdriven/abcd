---
schema_version: 1
id: "iss-2610262044029050"
slug: "the-dependency-graph-itd-78-lints-phase"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1179 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-78, itd-24]
finding_ref: cfk-55b233d7677a57da
---

The dependency graph (itd-78) lints phase ordering; itd-24 records that phases are retired

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "a phase that scopes an intent before its blocker's phase"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

One of the graph lints itd-78 promises fails when 'a phase scopes an intent before its blocker's phase', which assumes intents are scheduled into phases. itd-24 records that phases are retired in favour of releases, so the thing this lint checks is no longer part of the record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1179.
