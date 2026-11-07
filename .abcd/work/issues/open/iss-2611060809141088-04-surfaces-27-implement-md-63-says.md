---
schema_version: 1
id: "iss-2611060809141088"
slug: "04-surfaces-27-implement-md-63-says"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md"
remedy: "Add `implement stop` to the chapter's exceptions, saying it acts on the account's home or shared folder."
---

`04-surfaces/27-implement.md:63` says every sub-verb refuses outside a checkout or in a repository with no commit, with two exceptions, the load check and the report given a log file. `abcd implement stop` is a third: its `RunE` (`internal/surface/cli/implement_stop.go:198-213`) reads the caller, takes the stop home and sets or lifts the stop, and resolves no checkout.
