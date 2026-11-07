---
schema_version: 1
id: "iss-2611060809597822"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Add file writes through the host's own tools to the sentence."
---

`.abcd/development/brief/04-surfaces/17-guard.md:459-460` says the shim's `UNGUARDED` warning names shell commands or questions through the host's question tool. `hooks/pre-tool-use.sh:4` has a third wording for the host's file-writing tools, `file writes through the host's own tools run UNGUARDED`.
