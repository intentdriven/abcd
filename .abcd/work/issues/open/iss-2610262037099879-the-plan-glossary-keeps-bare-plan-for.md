---
schema_version: 1
id: "iss-2610262037099879"
slug: "the-plan-glossary-keeps-bare-plan-for"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 703 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/plan.md:51"
remedy: "Add a qualified row to the plan.md senses table for 'the drain plan': the classification and order a drain move derives from its snapshot of the ledger (DrainPlan in code, 'Show the plan first' in commands/drain.md), and say bare 'the plan' in 35-drain.md refers to it."
finding_ref: cfk-93b36b318e9a3fe9
---

The plan glossary keeps bare "plan" for the intent sign-off verb; the drain chapter uses bare "the plan" for a drain move's ordering of the ledger.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/plan.md:51` — "only for the intent verb and its sign-off. Qualify every other use"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:194` — "The plan, the lane's start check, the judgement's remedy digest and the reservation's footprint all read that snapshot"

plan.md says bare "plan" is used only for the intent verb and its sign-off, every other sense qualified, and never for the order of work. 35-drain.md writes "The plan", unqualified, for the classification and ordering a drain move derives from its snapshot of origin's ledger. The bare word carries a sense the glossary withholds from it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 703.
