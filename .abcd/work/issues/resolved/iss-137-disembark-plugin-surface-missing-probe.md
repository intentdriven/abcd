---
schema_version: 1
id: "iss-137"
slug: "disembark-plugin-surface-missing-probe"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "itd-88 fidelity review (2026-07-24 run queue, burst 10)"
found_at: "commands/abcd/disembark.md"
resolution: "commands/abcd/disembark.md now documents probe (coverage readout) and coverage (aggregate probe reports) sub-verbs; argument-hint updated"
impact: additive
---

probe and coverage ship as CLI verbs only: the plugin markdown surface (commands/abcd/disembark.md) documents plan and pack alone, while itd-88's press release headlines /abcd:disembark probe and coverage — either the surface doc grows the two verbs or the intent's diverged claim is the accepted record (dispositioned in itd-88's Audit Notes)