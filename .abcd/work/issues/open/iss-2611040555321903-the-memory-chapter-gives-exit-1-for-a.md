---
schema_version: 1
id: "iss-2611040555321903"
slug: "the-memory-chapter-gives-exit-1-for-a"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-memory.md"
remedy: "Scope the exit 1 statement to add and remove in both texts, and say the bare read and list exit 0 and name the file."
---

The memory chapter gives exit 1 for 'a machine file the read refuses' with no scoping to a sub-verb (.abcd/development/brief/04-surfaces/37-memory.md:97), and commands/memory.md:125-126 repeats it. Only `memory add` and `memory remove` exit 1 on such a file. Bare `abcd memory` and `abcd memory list` exit 0, name the file as unreadable and report no notes from it.
