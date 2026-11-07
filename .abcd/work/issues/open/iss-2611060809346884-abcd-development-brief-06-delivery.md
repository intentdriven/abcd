---
schema_version: 1
id: "iss-2611060809346884"
slug: "abcd-development-brief-06-delivery"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Add `shared-folder` to the list at 01-build-sequence.md:62 and make the count seven."
---

.abcd/development/brief/06-delivery/01-build-sequence.md:62 says six further forms of `ahoy` write: `uninstall`, `remote apply`, `connect`, `credential --home`, `worktree add` and `worktree prune`. `abcd ahoy shared-folder <folder>` is a seventh: it writes `shared-folder.json` in abcd's home folder (internal/core/runstop/runstop.go:344-360), and .abcd/development/brief/04-surfaces/01-ahoy.md:34 lists it as shipped.
