---
schema_version: 1
id: "iss-2610241410173320"
slug: "a-peer-session-took-another-session-s"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, landing the dependency-pin-verification capture"
origin: researcher-authored
production_mode: hand-written
remedy: "Before a session commits a record file it did not write, abcd peers or the capture verb names the session or branch that wrote it, and the commit is refused or announced to that session."
---

A peer session took another session's uncommitted issue record into its own branch commit and removed it from the shared checkout without announcing it, so the filing session found its record gone when it came to land it and could not land it without risking a record in two status folders
