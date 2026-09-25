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
resolution: "The DOGFOODING recall list names every top-level verb of abcd --help, and TestDogfoodingRecallNamesEveryTopLevelVerb holds the list to the root command's available verbs."
impact: internal
resolved_by:
  commit: "5f3a9eb6"
---

The DOGFOODING domain in .abcd/rules.json recalls on a fixed keyword list that has drifted from the verb tree: AGENTS.md says it injects the go-run rule on a prompt naming abcd or any of its top-level verbs (the Available Commands list), but decide, implement, inbox, mode, peers, report and statusline are missing from its recall, so a prompt naming one of them never receives the rule that keeps a session off the stale plugin-root binary. Nothing holds the list to the tree, which is why it drifted.

## Grounds

- pursued: we expect a verb added to the tree to fail go test until it joins the recall list, so the go-run rule reaches every prompt naming a verb; shown wrong if a verb in abcd --help is absent from the list while the suite is green
