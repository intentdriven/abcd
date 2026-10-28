---
schema_version: 1
id: "iss-2610252211582415"
slug: "readme-md-127-the-last-line-of-the"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D8"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Change README.md:127 from 'abcd v0.13.4' to 'abcd v0.15.1' and update it with each release cut, or make the sample's version line one that does not age. The labelled example in commands/abcd.md:24 can stay."
---

README.md:127, the last line of the sample board under the bare abcd prompt, reads 'abcd v0.13.4'. That line is the installed version (internal/core/board/board.go:138). The release a user installs today is v0.15.1: .claude-plugin/marketplace.json:15 names abcd-plugin-v0.15.1.zip, and v0.15.1 is the newest tag. So the sample shows a version two minor releases old. commands/abcd.md:24 uses the same version, as a labelled example.
