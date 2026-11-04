---
schema_version: 1
id: "iss-2611040554179196"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md"
remedy: "State the home ignore lay as an exception to the read-only claim."
duplicates: [iss-2611040554128376]
refines: [iss-2611040554149214]
---

.abcd/development/brief/04-surfaces/20-banlist.md:11-12 calls bare invocation and listing strictly read-only. Where ~/.abcd.noindex exists without its marker, the command first writes ~/.abcd.noindex/.gitignore and gitignore-laid (internal/surface/cli/homestop.go:176; .abcd/development/brief/05-internals/03-configuration.md:1202). Nothing is written when the folder is absent.
