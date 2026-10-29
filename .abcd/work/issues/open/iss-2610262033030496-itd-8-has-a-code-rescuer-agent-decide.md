---
schema_version: 1
id: "iss-2610262033030496"
slug: "itd-8-has-a-code-rescuer-agent-decide"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 353 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-8-with-code-bundling.md:31"
remedy: "In itd-8 drop the 'code-rescuer agent extended' scope bullet and say `pack --with-code` decides the code copy set deterministically (a rule over tracked files or a named path list, as mapping.go decides the rest); no agent runs inside pack."
related_intents: [itd-8]
finding_ref: cfk-d97b88475a6ed430
---

itd-8 has a code-rescuer agent decide what code the pack copies; the disembark page says pack is one deterministic run that dispatches no agents.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-8-with-code-bundling.md:31` — "`code-rescuer` agent extended (currently principle-only) to make scope-aware copy decisions"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes"

itd-8 says that `disembark pack --with-code` packs the synthesised brief, principles and reviews along with the code, and that a `code-rescuer` agent, described as currently principle-only, makes the scope-aware copy decisions. The disembark page says packing is one deterministic Go run that dispatches no agents and runs no model passes. The principles, press release and review are written afterwards by the synthesis sub-verbs and do not exist at pack time. Its agent roster has no code-rescuer: principles go to principle-distiller. An agent choosing the copy set inside pack cannot coexist with a pack that dispatches no agents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 353.
