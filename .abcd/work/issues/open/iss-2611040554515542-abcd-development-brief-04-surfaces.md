---
schema_version: 1
id: "iss-2611040554515542"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Add the session commands to the chapter's list and correct the command page's claim."
---

.abcd/development/brief/04-surfaces/21-update.md:26-30 names the launch preview and the release cut as the other commands reaching the release origin, and commands/update.md:18 calls --fetch-baseline the one other flag that does. `abcd session blueprint` and `abcd session setup` also read a release's checksums.txt from that origin (internal/surface/cli/session.go:70, 534, 798; .abcd/development/brief/04-surfaces/37-session.md:123).
