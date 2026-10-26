---
schema_version: 1
id: "iss-2610262039384176"
slug: "itd-146-groups-spec-drain-and-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 875 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-cc9880a86bdbcfb6
---

itd-146 groups `spec`, `drain` and `memory` under records, `lint` under checks and `changelog` in the agent block; the surfaces register's help table puts those verbs with the agents and `changelog` in neither block

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:168` — "records (`capture`, `intent`, `spec`, `decide`, `build`, `drain`, `memory`)"
- End B: `.abcd/development/brief/04-surfaces/README.md:148` — "`build`, `capture`, `decide`, `ideate`, `intent`, `land`, `library`, `status`"

itd-146's ruled groups put `capture`, `intent`, `spec`, `decide`, `build`, `drain` and `memory` under records, put `lint` under checks, and keep `changelog` in the agent block. The surfaces register's help table lists records as `build`, `capture`, `decide`, `ideate`, `intent`, `land`, `library` and `status`, and checks as `lab` and `reading`. It lists `spec`, `drain`, `lint` and `memory` among the agents' verbs, and says `changelog` is in neither block because it merged into `launch`. The two documents assign the same verbs to different groups.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 875.
