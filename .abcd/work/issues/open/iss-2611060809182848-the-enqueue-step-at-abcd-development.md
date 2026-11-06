---
schema_version: 1
id: "iss-2611060809182848"
slug: "the-enqueue-step-at-abcd-development"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-land.md"
remedy: "Add the stacked-branch deferral and its two ways out to the enqueue step in 38-land.md."
---

The enqueue step at .abcd/development/brief/04-surfaces/38-land.md:106 says the pull request is armed where the merge gate allows it and names no other hold. internal/core/landing/stacked.go also defers the arming where a merge queue gates the default branch and the branch shares a commit not on it with another queued or armed pull request, and the step then stops naming it. `abcd land --help` documents this; the chapter does not.
