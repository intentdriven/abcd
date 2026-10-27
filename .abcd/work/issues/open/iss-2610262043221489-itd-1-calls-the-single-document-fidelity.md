---
schema_version: 1
id: "iss-2610262043221489"
slug: "itd-1-calls-the-single-document-fidelity"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1133 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-34]
finding_ref: cfk-ea514167952d6b9a
---

itd-1 calls the single-document fidelity auditor intent-auditor; itd-34 calls the same agent intent-fidelity-reviewer

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17` — "emitted by the `intent-auditor` agent's single-document role"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "`intent-fidelity-reviewer` shape-classification role (third role)"

itd-1 says per-criterion verdicts are emitted by the intent-auditor agent's single-document role, and itd-34 says the agent that performs single-document fidelity audits per itd-1 is intent-fidelity-reviewer, which gains a third role. One agent carries two names across the two records, so a reader cannot tell whether the shape-classification role belongs to the auditor the verdict pipeline dispatches.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1133.
