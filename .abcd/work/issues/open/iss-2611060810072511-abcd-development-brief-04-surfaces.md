---
schema_version: 1
id: "iss-2611060810072511"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Add the drain verb to both sentences as a way a pending transcript gets placed."
---

`.abcd/development/brief/04-surfaces/11-history.md:208-210` says a pending transcript the session start cannot place stays pending and unredacted until a later start can place it or an operator discards it, and `.abcd/development/brief/04-surfaces/11-history.md:95` describes draining only as redacting and storing staged transcripts. `abcd history drain` also places pending transcripts first (`placePendingNotices`, `internal/surface/cli/history.go:392`), so one can be filed and stored with no session start. `commands/history.md:66` states this; the chapter does not.
