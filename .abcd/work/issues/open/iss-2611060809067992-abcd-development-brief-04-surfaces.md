---
schema_version: 1
id: "iss-2611060809067992"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Add the five entries to the tree in 01-ahoy.md and to the prose in 03-configuration.md."
---

.abcd/development/brief/04-surfaces/01-ahoy.md:631 says its inventory of abcd's home folder and the one in 05-internals/03-configuration.md are one list. Neither names `agents/` (internal/core/runner/ledger.go:49), `profiles/` (internal/core/runner/profile.go:50), or `shared-folder.json`, `stop.json` and `stop-log.jsonl` (internal/core/runstop/runstop.go:44-54), all of which the binary keeps there. The missing `fill/` entry is held by iss-2611040554089130.
