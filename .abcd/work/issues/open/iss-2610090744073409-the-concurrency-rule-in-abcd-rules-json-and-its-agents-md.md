---
schema_version: 1
id: "iss-2610090744073409"
slug: "the-concurrency-rule-in-abcd-rules-json-and-its-agents-md"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "2026-10-09 ideate research leg"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
remedy: "Change 'in drafts/' to 'planned' in the CONCURRENCY rule text and anywhere else that repeats it, and let the rule say the store has no verbs yet because the planned intent has not shipped."
---

The CONCURRENCY rule in .abcd/rules.json (and its AGENTS.md short form, if it repeats it) says the worktree store's own verbs are itd-2609091014076309, in drafts/, but that intent is in planned/; its link already points at planned/, so the prose contradicts the link beside it.
