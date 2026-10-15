---
schema_version: 1
id: "iss-2610151422129661"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:557 to match the shipped surface: The shipped `abcd memory` verb (commands/memory.md; sub-verbs add, list, remove) keeps personal memory notes in a machine layer at ~/.abcd.noindex/memory.json (internal/core/memnote/layers.go MachineRelPath), written by `abcd memory add` (with --everywhere…"
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:557 says: User scope holds 'user-scope memory for personal cross-project knowledge (a later phase too: the shipped memory store is repo-scope)'; ':1023 the user-scope `~/.abcd.noindex/memory/` is a later phase ... (nothing in the binary resolves it today)'. The shipped surface differs: The shipped `abcd memory` verb (commands/memory.md; sub-verbs add, list, remove) keeps personal memory notes in a machine layer at ~/.abcd.noindex/memory.json (internal/core/memnote/layers.go MachineRelPath), written by `abcd memory add` (with --everywhere for cross-repo recall), plus a repo layer .abcd/memory.json; `abcd hook prompt-router` recalls them. Neither memory.json file appears in the user-scope or repo-scope inventory. (v0.14.2 release crosscheck, false-claim, raised by opus in a34.)
