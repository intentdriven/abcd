---
schema_version: 1
id: "iss-2611060810055166"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Describe supersession, its prefix no-op and the `superseded` field in the chapter and the command page."
---

`.abcd/development/brief/04-surfaces/11-history.md:55-58` describes capture as idempotent on the content hash, session id and source labels together: identical content is a no-op and a different session writes a new record. Capture also supersedes (`resolveSupersession`, `internal/core/history/history.go:391`). A body that extends a stored record of the same session, agent and labels deletes that record and the result carries a `superseded` field, and a body that is a prefix of a stored one is a no-op despite its different hash. Neither the chapter nor `commands/history.md` says this, or that a capture can remove a stored record.
