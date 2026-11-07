---
schema_version: 1
id: "iss-2611060809092034"
slug: "04-surfaces-34-build-md-295-says-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md"
remedy: "Raise the chapter's version statements to 21 and add version 21's two fields to its history."
---

`04-surfaces/34-build.md:295` says the state is schema version 20, and line 383 says an earlier version is written back at version 20 by its next mutation. `internal/core/implement/loop/state.go:281` declares `SchemaVersion = 21`, and state.go:285 keeps 20 as read and never written. Version 21 added a round's `resolves_shown` and a landing's `gate_unread_since` (state.go:271-276), which no brief chapter names.
