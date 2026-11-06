---
schema_version: 1
id: "iss-2611060809013416"
slug: "04-surfaces-readme-md-191-says-every"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Name the framework commands' exemption and its decisions in the sentence section of the surfaces chapter."
---

`04-surfaces/README.md:191` says every visible verb and sub-verb carries the one sentence, and line 214 says a test walks every visible command. `abcd help`, `abcd completion` and its shell sub-commands are visible, carry the framework's own text, and are exempted by name at `internal/surface/cli/sentences_test.go:84-90`. The section states no exemption, though line 265 does for the separate bare-render test.
