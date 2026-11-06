---
schema_version: 1
id: "iss-2611060809557633"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Name the lock file among what embark leaves, or have embark make sure the target ignores it."
---

`.abcd/development/brief/04-surfaces/03-embark.md:135-136` says embark writes the record families plus the marker block. An embark that writes an issue also leaves `.abcd/work/issues/.iss-alloc.lock` in the target, from the ledger lock taken at `internal/core/lifeboat/embark.go:157`. `.abcd/development/brief/04-surfaces/03-embark.md:121` names the lock but not the file, and in a fresh target the file shows as untracked, so a plain `git add` of the result commits it.
