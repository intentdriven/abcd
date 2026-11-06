---
schema_version: 1
id: "iss-2611060809036916"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/41-agents.md"
remedy: "Name spc-2611050023402644 as the spec holding the two remaining steps."
---

`.abcd/development/brief/04-surfaces/41-agents.md:60-61` says the status line's row and an unattended run's summary are "the spec's remaining steps", and the only spec the chapter names (line 4) is spc-2611041701500441, which is in `.abcd/development/specs/closed/`. Those steps moved to the open remainder spc-2611050023402644, which the chapter does not name.
