---
schema_version: 1
id: "iss-2610262045198472"
slug: "itd-67-picks-the-release-bump-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1266 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67, itd-24]
finding_ref: cfk-d231d328ae6fadb6
---

itd-67 picks the release bump by phase-completion detection; itd-24 records that phases are retired and the release is the unit

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "`/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

itd-67's `launch ship` auto-selects the bump tier through the brief §4 phase-completion detection. itd-24 records the ruling (adr-2609212115255771) that phases are retired, with a phase read as a release tag. A bump that depends on detecting phase completion has nothing left to detect.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1266.
