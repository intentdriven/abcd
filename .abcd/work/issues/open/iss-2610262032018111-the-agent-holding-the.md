---
schema_version: 1
id: "iss-2610262032018111"
slug: "the-agent-holding-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 210 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:72"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-6ed63ee19e936386
---

The agent holding the shape-classification third role is `intent-auditor` in the mental model and `intent-fidelity-reviewer` in itd-34.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:72` — "A continuous audit role (the third role on `intent-auditor`"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "`intent-fidelity-reviewer` shape-classification role (third role)"

Both documents describe the same continuous third role that suggests reclassifications, but the brief puts it on `intent-auditor` and itd-34 puts it on `intent-fidelity-reviewer`. That is two names for one agent, and a reader cannot tell from the record whether they are the same agent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 210.
