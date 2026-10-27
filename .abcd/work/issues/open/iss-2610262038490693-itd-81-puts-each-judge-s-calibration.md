---
schema_version: 1
id: "iss-2610262038490693"
slug: "itd-81-puts-each-judge-s-calibration"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 820 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-81]
finding_ref: cfk-911a73ce9cdc3f57
---

itd-81 puts each judge's calibration corpus at agents/< name>/corpus/; 01-agents says agents/ holds prompts and fixture directories and nothing else

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:25` — "`agents/<name>/corpus/`, containing labelled cases of **both** classes:"
- End B: `.abcd/development/brief/05-internals/01-agents.md:68` — "`agents/` holds the prompt files and their per-agent fixture directories and nothing else."

itd-81 requires each verdict-rendering agent to have a calibration corpus at agents/< name>/corpus/. 01-agents says agents/ holds the prompt files and their per-agent fixture directories and nothing else. The corpus location the discipline requires is excluded by the agent-directory contract.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 820.
