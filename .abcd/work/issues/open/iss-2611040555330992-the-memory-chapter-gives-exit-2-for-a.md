---
schema_version: 1
id: "iss-2611040555330992"
slug: "the-memory-chapter-gives-exit-2-for-a"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/memory.go"
remedy: "Map a gate that could not judge the note to exit 2, or state in the chapter and the page that those two refusals exit 1."
---

The memory chapter gives exit 2 for a refusal by the gate (.abcd/development/brief/04-surfaces/37-memory.md:98), its gate section lists a degraded scanner and an unreadable private-names store among the refusals, and commands/memory.md:125 says every refusal exits 2. Both of those refuse the note, write nothing and exit 1, because only memnote.GateError and the named field and record errors map to 2 (internal/surface/cli/memory.go:143-147). A banlist match and a scanner span do exit 2.
