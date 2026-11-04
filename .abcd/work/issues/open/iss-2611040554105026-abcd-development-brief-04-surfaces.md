---
schema_version: 1
id: "iss-2611040554105026"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md"
remedy: "Say setup offers to set it at a terminal, and that off a terminal abcd prints the command."
---

.abcd/development/brief/04-surfaces/20-banlist.md:247-250 says git runs the hook the clone's hooks path selects, which abcd neither sets nor fully observes. On a yes at a terminal `abcd ahoy install` sets core.hooksPath to .githooks in the clone's .git/config and reads it back (internal/core/ahoy/commit_checks.go:44-47; .abcd/development/brief/04-surfaces/01-ahoy.md:984).
