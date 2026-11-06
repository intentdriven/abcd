---
schema_version: 1
id: "iss-2611060809172280"
slug: "04-surfaces-38-land-md-352-lists-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/38-land.md"
remedy: "Add `withdraw` to the chapter's list of step names, in its place before `push`."
---

`04-surfaces/38-land.md:352` lists the steps in the JSON as `origin` when it reports a stale default, `gate`, `announce`, `push` and `enqueue`, or `land` alone. `internal/core/landing/withdraw.go:237,272,276,292` adds a further step named `withdraw`, with status `left` or `done`. Lines 98-103 of the chapter describe the act but the list does not name its step.
