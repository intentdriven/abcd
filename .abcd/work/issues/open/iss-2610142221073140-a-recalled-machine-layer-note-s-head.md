---
schema_version: 1
id: "iss-2610142221073140"
slug: "a-recalled-machine-layer-note-s-head"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "spc-2610111627213138 security round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recall/render.go"
remedy: "Render a machine note's origin in its label (for example \"personal note, moved from the host's memory\" for origin host-memory-move), keeping \"agent-written\" for origin agent."
---

A recalled machine-layer note's head reads "(personal note, agent-written)" whatever its origin, so a note moved in from the host's own memory with memory add --moved-from-host carries the same label as one an agent wrote of its own accord. The label is true of how the file is written (an agent runs memory add) but not of where the lesson came from, which the note's origin field records.
