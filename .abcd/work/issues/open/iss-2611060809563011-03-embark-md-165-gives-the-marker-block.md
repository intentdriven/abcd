---
schema_version: 1
id: "iss-2611060809563011"
slug: "03-embark-md-165-gives-the-marker-block"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Add the unreadable settings file to the marker outcomes and name how it differs from an unknown value."
---

`03-embark.md:165` gives the marker block's outcomes by `docs.target` and does not list an unreadable `.abcd/config.json`, which skips the block with the note `could not be read, so abcd's block was not planted; fix that file and run embark again` (`markerSkipUnreadable`, `internal/core/lifeboat/embark.go:578`). The chapter also does not say this differs from an unknown value, which reads as no choice and goes to `AGENTS.md`.
