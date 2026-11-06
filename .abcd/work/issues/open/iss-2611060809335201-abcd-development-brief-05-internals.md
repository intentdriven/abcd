---
schema_version: 1
id: "iss-2611060809335201"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Restate why the harness waits for the first Pass-A spec, or drop the clause about a second agent."
---

`.abcd/development/brief/05-internals/05-prompt-quality.md:81-82` says the layer-B harness lands with the first Pass-A agent's spec, the point at which a second agent exists to generalise the runner over. Eighteen prompts already ship under `agents/`, as the same section counts at lines 74-75, and no Pass-A agent has shipped (`.abcd/development/brief/05-internals/01-agents.md:76-88`), so the reason given for the timing no longer holds.
