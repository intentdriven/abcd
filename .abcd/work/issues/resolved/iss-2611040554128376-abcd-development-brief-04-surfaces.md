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
resolution: "The board chapter (04-surfaces/08-abcd.md) now states the home ignore lay as the one exception to its read-only claims: where ~/.abcd.noindex exists without its gitignore-laid marker, the CLI front door writes ~/.abcd.noindex/.gitignore and the marker before every command but the status line, nothing in the repository and nothing at all where the folder is absent or the marker stands. It cites the internals section (05-internals/03-configuration.md, The two .abcd/ scopes), and the opening and version passages point at the statement."
impact: internal
resolved_by:
  commit: "382c488784a389d0a7cfbc880b849cc35e1acb70"
---

.abcd/development/brief/04-surfaces/08-abcd.md:5-7, 37 and 465-466 call the board strictly read-only and say the version answer writes nothing. Where ~/.abcd.noindex exists without its marker, every command but the status line first writes ~/.abcd.noindex/.gitignore and gitignore-laid (internal/surface/cli/cli.go:7247, layHomeIgnore in internal/surface/cli/homestop.go:176; .abcd/development/brief/05-internals/03-configuration.md:1202). Nothing is written in the repository, and nothing at all when the folder is absent.

## Grounds

- pursued: a reader of the board chapter learns of the one write its read-only forms can make and where the internals describe it; a read-only or writes-nothing claim in the chapter's opening, What ships today or The version that names no exception, or a link that no longer reaches the internals passage, would show it wrong
