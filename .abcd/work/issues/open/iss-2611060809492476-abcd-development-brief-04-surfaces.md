---
schema_version: 1
id: "iss-2611060809492476"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "Limit the skip to the bare verb in the chapter and the command page, and name the drive's refusal beside it."
---

`.abcd/development/brief/04-surfaces/35-drain.md:759-760` says a route of the judge to opencode is skipped with a diagnostic and the judge stays on the host, with no exception for the drive, and `commands/drain.md:384-385` says the same. Under `--drive` that route refuses the start: `drivePreconditions` returns the refusal at `internal/surface/cli/drain_drive.go:246-247`, and `abcd drain --help` lists it. The skip holds for the bare verb only.
