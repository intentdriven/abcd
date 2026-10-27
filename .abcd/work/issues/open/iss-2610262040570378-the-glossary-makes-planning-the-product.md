---
schema_version: 1
id: "iss-2610262040570378"
slug: "the-glossary-makes-planning-the-product"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 966 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/plan.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-2609201916151817]
finding_ref: cfk-defe297e14e1f8c4
---

The glossary makes planning the product thinker's sign-off and never unattended; the build intent adds `--auto-plan`, which plans with no human.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/plan.md:19` — "sign-off on an intent's acceptance criteria — never run unattended"
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:68` — "it lets the planning path run without a human"

plan.md defines moving a draft to planned/ as the product thinker's sign-off on the acceptance criteria, never run unattended and never inferred. itd-2609201916151817 decision 6 adds an agent path, `implement --auto-plan`, that takes a draft through planning with no human and replaces the sign-off with recorded decisions and review receipts. The two documents disagree on whether planning can ever happen without the sign-off.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 966.
