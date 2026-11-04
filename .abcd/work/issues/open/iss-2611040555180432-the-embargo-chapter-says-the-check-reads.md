---
schema_version: 1
id: "iss-2611040555180432"
slug: "the-embargo-chapter-says-the-check-reads"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-embargo.md"
remedy: "Add the names of files a commit adds to the range check's enumeration in the chapter and the help."
refines: [iss-2610150526192138]
---

The embargo chapter says the check reads, over a commit range, messages and added lines (.abcd/development/brief/04-surfaces/37-embargo.md:86-87), and `abcd embargo check --help` says the same. The check also scans the names of files a commit adds, reporting 'a file name the commit adds' (internal/core/embargo/check.go:316). commands/embargo.md:175 states this; the chapter and the help do not.
