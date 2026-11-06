---
schema_version: 1
id: "iss-2611060809438042"
slug: "abcd-development-brief-05-internals"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md"
remedy: "State that recall reads both scopes' pages directly and suppresses by filename, or implement the keyword rule the chapter describes."
---

`.abcd/development/brief/05-internals/07-memory.md:193` says the hook queries the `index.md` of both memory scopes and that on a recall-keyword collision across scopes the narrower scope wins, repo overriding user. `index.md` is never read (`internal/core/memory/recallpages.go:27`, `:61`). A machine page is dropped only when a repo page has the identical filename (`pageItems`, `internal/core/recall/pages.go:22`), so a repo page and a machine page with different filenames and the same `recall:` word are both recalled for one prompt.
