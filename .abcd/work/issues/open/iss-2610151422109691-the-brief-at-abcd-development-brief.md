---
schema_version: 1
id: "iss-2610151422109691"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite .abcd/development/brief/02-constraints/04-naming.md:225 to match the shipped surface: Only `disembark/history.jsonl` is written (internal/core/lifeboat/voyage.go joins only `<root-sha>/disembark`); nothing in internal/ writes `embark/provenance.json` or `embark/from/`, and 04-surfaces/03-embark.md §7 marks the `embark/` subtree 'not built…"
---

The brief at .abcd/development/brief/02-constraints/04-naming.md:225 says: `voyage/` row (Type column, unmarked): the operations namespace is 'Split by operation: `disembark/history.jsonl`, `embark/provenance.json`, `embark/from/<timestamp>/`'. The shipped surface differs: Only `disembark/history.jsonl` is written (internal/core/lifeboat/voyage.go joins only `<root-sha>/disembark`); nothing in internal/ writes `embark/provenance.json` or `embark/from/`, and 04-surfaces/03-embark.md §7 marks the `embark/` subtree 'not built yet'. The row carries no **(staged)** mark, which the file's own truth rule (line 156) requires for a design target. (v0.14.2 release crosscheck, false-claim, raised by opus in a32.)
