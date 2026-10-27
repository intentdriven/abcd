---
schema_version: 1
id: "iss-2610262042486043"
slug: "itd-142-states-abcd-ships-zero-skills"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1094 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-142, itd-44]
finding_ref: cfk-7ca3b7e3c8e0d0f7
---

itd-142 states abcd ships zero skills; itd-44 routes capture through an abcd-intent-new skill

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:13` — "abcd ships zero skills"
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111` — "the `abcd-intent-new` skill is wrapper-only"

itd-142 makes the brief interview a command because abcd ships zero skills. itd-44's capture routing depends on the abcd-intent-new skill calling the CLI and then routing. One record says no skill exists; the other's design rests on one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1094.
