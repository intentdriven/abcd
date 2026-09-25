---
schema_version: 1
id: "iss-2609251357387577"
slug: "the-dogfooding-domain-in-abcd-rules-json-recalls-on-a-fixed"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
---

The DOGFOODING domain in .abcd/rules.json recalls on a fixed keyword list that has drifted from the verb tree: AGENTS.md says it injects the go-run rule on a prompt naming abcd or any of its top-level verbs (the Available Commands list), but decide, implement, inbox, mode, peers, report and statusline are missing from its recall, so a prompt naming one of them never receives the rule that keeps a session off the stale plugin-root binary. Nothing holds the list to the tree, which is why it drifted.
