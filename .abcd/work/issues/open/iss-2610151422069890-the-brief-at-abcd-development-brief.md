---
schema_version: 1
id: "iss-2610151422069890"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/08-abcd.md:413-417 to match the shipped surface: `memory` is a registered, shipped top-level verb (`abcd memory --help`: 'Count the memory notes in both layers...'; listed under 'For agents and hosts' in `abcd --help --agent`; commands/memory.md exists; brief README row 37 marks `/abcd:memory` shipped).…"
---

The brief at .abcd/development/brief/04-surfaces/08-abcd.md:413-417, .abcd/development/brief/04-surfaces/08-abcd.md:416 says: A name retired with no alias is answered with what replaced it, and never as a stale binary: ... `memory` and `source`, which merged into `library`. The shipped surface differs: `memory` is a registered, shipped top-level verb (`abcd memory --help`: 'Count the memory notes in both layers...'; listed under 'For agents and hosts' in `abcd --help --agent`; commands/memory.md exists; brief README row 37 marks `/abcd:memory` shipped). Only `source` is a retired name: `abcd source` prints the merged-into-library line, while `abcd memory` runs. The chapter lists a live verb as retired. (v0.14.2 release crosscheck, false-claim, raised by opus+sonnet in a07.)
