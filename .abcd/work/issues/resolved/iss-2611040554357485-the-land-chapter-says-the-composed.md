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
resolution: "The land chapter now says the composed pull-request body leaves merge commits out and keeps, once each, only a Resolves: or Refs: line naming issue ids alone and a Delivers: line naming intent ids alone; every other such line is dropped, Refs: spc-9 step 3, Refs: itd-N and Refs: #12 among them."
impact: internal
resolved_by:
  commit: "e01291571428ccb551a1571c53a40b09479d3fe9"
---

The land chapter says the composed pull-request body carries the commits' Refs:, Resolves: and Delivers: lines once each (.abcd/development/brief/04-surfaces/38-land.md:143). The composer keeps only Resolves: or Refs: lines naming iss-N ids alone and Delivers: lines naming itd-N ids alone (internal/core/landing/prbody.go:46-47,82), and leaves merge commits out (prbody.go:62). `Refs: spc-9 step 3`, the chapter's own example at line 69, is dropped, as are `Refs: itd-N` and `Refs: #12`.

## Grounds

- pursued: the chapter's composed-body passage matches what internal/core/landing/prbody.go keeps (its two line patterns and --no-merges); a body composed from a branch carrying Refs: spc-9 step 3 or a merge commit that showed either in the body would show the passage wrong. The detector (detector/chapter_test.go in the lane directory, failed before, passes after) is not committed, because the widen to a test file was refused for want of a push credential; make preflight was cut off by the harness, so only make precheck ran to the end.
