---
schema_version: 1
id: "iss-2611040554128376"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "State the home ignore lay as the one exception to the chapter's read-only claims, citing the internals section."
---

.abcd/development/brief/04-surfaces/08-abcd.md:5-7, 37 and 465-466 call the board strictly read-only and say the version answer writes nothing. Where ~/.abcd.noindex exists without its marker, every command but the status line first writes ~/.abcd.noindex/.gitignore and gitignore-laid (internal/surface/cli/cli.go:7247, layHomeIgnore in internal/surface/cli/homestop.go:176; .abcd/development/brief/05-internals/03-configuration.md:1202). Nothing is written in the repository, and nothing at all when the folder is absent.
