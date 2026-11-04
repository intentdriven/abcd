---
schema_version: 1
id: "iss-2611040554139508"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Add `ceiling` to the chapter's enumeration of the JSON form, pointing at the command page for its fields."
---

.abcd/development/brief/04-surfaces/08-abcd.md:286-320 enumerates what `abcd peers --json` adds and does not name the top-level `ceiling` object (internal/surface/cli/peers.go:72, internal/core/claim/ceiling.go:51), which is emitted when the reader's identity is known. commands/peers.md:120 documents its fields; .abcd/development/brief/04-surfaces/37-reserve.md describes the cap only as behaviour.
