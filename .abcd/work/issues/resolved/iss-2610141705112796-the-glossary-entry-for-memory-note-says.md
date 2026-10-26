---
schema_version: 1
id: "iss-2610141705112796"
slug: "the-glossary-entry-for-memory-note-says"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/memory-note.md"
remedy: "Say the command is planned under itd-2610091918433290, or describe the note without a command until it ships."
refines: [iss-2609091956001547, iss-35, iss-2608231346137587]
resolution: "The memory verb now ships: c8d1ab0df added abcd memory add, and commands/memory.md documents it, so the glossary entry that says an agent writes a note with /abcd:memory add describes shipped behaviour."
impact: internal
resolved_by:
  commit: "c8d1ab0df"
---

The glossary entry for memory note says an agent writes one with /abcd:memory add, as if shipped; no memory verb or command exists, and memory notes are an open intent (itd-2610091918433290).

## Grounds

- pursued: we expect the memory-note glossary entry to stay accurate as written, because the verb it names ships and the prompt recall reads its notes; shown wrong if abcd memory add is removed or renamed while the entry still names it.
