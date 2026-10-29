---
schema_version: 1
id: "iss-2610262032566420"
slug: "itd-5-applies-its-rule-to-a-pass-a-spine"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 337 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25"
remedy: "In itd-5 line 25 (and the canary list at line 66 and acceptance criteria at lines 119-120), replace the Pass A spine / chat-distiller / embark-scaffolder roster with every prompt in agents/ today, and describe the unbuilt design-roster agents as future specs the rule will bind"
related_intents: [itd-5]
finding_ref: cfk-378a76eababddbfe
---

itd-5 applies its rule to a Pass A spine agent spec; disembark packs the spine in one deterministic run that dispatches no agents

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25` — "Pass A spine, Pass B chat-distiller, Pass C principle / artefact / brief / press-release agents"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes"

itd-5's rule lists the agent specs it governs, beginning with a Pass A spine agent. The disembark chapter says packing, which writes the rescue/spine.md commit-history spine, dispatches no agents and runs no model passes. Its only agents are the ones the later synthesis sub-verbs delegate to. The discipline's scope names a lifeboat agent that the lifeboat surface says never runs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 337.
