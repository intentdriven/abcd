---
schema_version: 1
id: "iss-2610262045168363"
slug: "the-agent-that-grades-shipped-intents-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1263 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-48, itd-24]
finding_ref: cfk-a4025c3fd459caec
refines: [itd-48]
---

The agent that grades shipped intents is `intent-fidelity-reviewer` in itd-48 and 'the intent auditor' in itd-24

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "abcd's `intent-fidelity-reviewer` agent grows from one `role` to three."
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:61` — "judging delivered reality against each intent's criteria is the intent auditor's job"

itd-48 names the agent that grades shipped intents, and gains the consistency and kind roles, `intent-fidelity-reviewer`. itd-24 gives the same job, judging delivered reality against each intent's criteria, to the intent auditor. That is two names for one agent definition, so a reader cannot tell which file holds the role.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1263.
