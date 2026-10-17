---
schema_version: 1
id: "iss-2610151422061914"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/07-library.md:11-16 to match the shipped surface: `source` is gone from the binary, but `memory` is still a registered top-level verb (`abcd memory --help`: \"Count the memory notes in both layers\"). It now addresses a different store, the memory notes in `~/.abcd.noindex/memory.json` and `.abcd/memory.json`,…"
---

The brief at .abcd/development/brief/04-surfaces/07-library.md:11-16 says: The library command merges `source` and `memory`, and under adr-40 "neither old name survives as an alias". The chapter then calls `.abcd/memory/` "the memory store" throughout and never mentions another memory surface. The shipped surface differs: `source` is gone from the binary, but `memory` is still a registered top-level verb (`abcd memory --help`: "Count the memory notes in both layers"). It now addresses a different store, the memory notes in `~/.abcd.noindex/memory.json` and `.abcd/memory.json`, and is documented in 37-memory.md. Strictly it is not an alias, but the old name does survive on the verb tree with a different meaning. This chapter does not mention it or tell the two memory stores apart. (v0.14.2 release crosscheck, false-claim, raised by opus in a06.)
