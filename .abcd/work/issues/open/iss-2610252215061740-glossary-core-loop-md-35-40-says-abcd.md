---
schema_version: 1
id: "iss-2610252215061740"
slug: "glossary-core-loop-md-35-40-says-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md"
remedy: "Add the implement loop as a qualified sense, and restate the run-loop sense as a seam whose default backend is abcd's native loop, in both files."
---

glossary/core/loop.md:35-40 says 'abcd never owns the run loop ... a pluggable adapter supplies the loop itself'; the run seam's native default is a 'thin native Go loop (adr-27)' (05-internals/02-adapters.md:25) and the implement loop is abcd's own loop over a state file (04-surfaces/34-build.md:5-7). The Senses table also lacks the implement/build-loop sense step.md:18 and oracle.md:25 use. docs/reference/terminology.md:45 repeats 'never owns the loop'.
