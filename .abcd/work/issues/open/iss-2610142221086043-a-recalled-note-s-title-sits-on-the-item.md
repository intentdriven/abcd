---
schema_version: 1
id: "iss-2610142221086043"
slug: "a-recalled-note-s-title-sits-on-the-item"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "spc-2610111627213138 security round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recall/render.go"
remedy: "Render the title as literal text, inside a code span sized by the tree's one code-span rule or inside the body's fence, so no inline construct in a project note's title renders."
---

A recalled note's title sits on the item's head line, outside the body's fence. It is one validated, masked line placed after the id and the layer label, so it cannot open a heading, a list item or a fence, but inline markdown in it (a link, emphasis, a code span, inline HTML) is still live to a markdown reader. A project note's title comes from the repository's committed .abcd/memory.json, so a cloned or compromised repository controls how it renders.
