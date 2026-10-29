---
schema_version: 1
id: "iss-2610262032578385"
slug: "itd-15-has-every-disembark-of-abcd-cli"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 341 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-15-self-dogfooded-sota-audit.md:16"
remedy: "Re-scope itd-15 as an explicit, operator-run synthesis sub-verb over an already-packed lifeboat (or a developer-side prompt audit), not a Pass C that runs automatically inside pack; drop 'automatically' and the three-pass wording."
related_intents: [itd-15]
finding_ref: cfk-113d63e32ac5298b
---

itd-15 has every disembark of abcd-cli run an agent audit pass automatically; disembark's pack dispatches no agents, and synthesis runs only as later sub-verbs

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-15-self-dogfooded-sota-audit.md:16` — "Pass C automatically runs a SOTA-prompt-audit pass over every `agents/*.md` file in the repo"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes"

itd-15 makes a Pass C agent run automatically whenever abcd-cli is disembarked, with its findings shipped in the lifeboat. The disembark chapter says packing dispatches no agents and runs no model passes. Synthesis artefacts are written afterwards by explicit sub-verbs over an already-packed lifeboat. An automatic agent pass inside disembark breaks that architecture.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 341.
