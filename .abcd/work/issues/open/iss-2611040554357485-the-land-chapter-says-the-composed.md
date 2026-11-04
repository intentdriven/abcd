---
schema_version: 1
id: "iss-2611040554357485"
slug: "the-land-chapter-says-the-composed"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-land.md"
remedy: "Say in the chapter which trailer lines the body keeps, those naming only issue or intent ids, and that merge commits are left out."
---

The land chapter says the composed pull-request body carries the commits' Refs:, Resolves: and Delivers: lines once each (.abcd/development/brief/04-surfaces/38-land.md:143). The composer keeps only Resolves: or Refs: lines naming iss-N ids alone and Delivers: lines naming itd-N ids alone (internal/core/landing/prbody.go:46-47,82), and leaves merge commits out (prbody.go:62). `Refs: spc-9 step 3`, the chapter's own example at line 69, is dropped, as are `Refs: itd-N` and `Refs: #12`.
