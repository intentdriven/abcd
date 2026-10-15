---
schema_version: 1
id: "iss-2610151422122793"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:847 to match the shipped surface: `abcd hook prompt-router --help`: plain stdout is the rule block 'followed by a \"## MEMORY\" block of the memory notes the prompt recalls', and the --json envelope is {\"text\", \"injected\", \"recalled\", \"active\", \"error\"}; the chapter does not mention the memory…"
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:847 says: prompt-router stdout carries 'the rendered block of the domains new this turn, and nothing else'; the --json envelope's fields are text, injected, active and error. The shipped surface differs: `abcd hook prompt-router --help`: plain stdout is the rule block 'followed by a "## MEMORY" block of the memory notes the prompt recalls', and the --json envelope is {"text", "injected", "recalled", "active", "error"}; the chapter does not mention the memory block or the `recalled` field. (v0.14.2 release crosscheck, false-claim, raised by opus in a34.)
