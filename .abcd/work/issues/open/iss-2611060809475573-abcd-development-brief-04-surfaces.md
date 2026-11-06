---
schema_version: 1
id: "iss-2611060809475573"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "State the drive's exception in the chapter and the help: a drive gives a live drain with no cap the lanes opened plus `--max`, or plus eight."
---

`.abcd/development/brief/04-surfaces/35-drain.md:654-655` says the cap is set when a drain begins and a different cap named while it runs is refused, and `.abcd/development/brief/04-surfaces/35-drain.md:943` repeats the refusal. That holds for the bare verb and for a drive over a drain that already has a cap. A drive over a live drain with no cap gives it one: `internal/surface/cli/drain_drive.go:327` refuses a differing `--max` only when the drain's cap is above zero, and `internal/surface/cli/drain_drive.go:391` calls `loop.CapDrainInProgress` (`internal/core/implement/loop/terminal.go:1855`), which stores the lanes already opened plus `--max`, or plus the default of eight. The help at `internal/surface/cli/drain.go:235-236` repeats the chapter's sentence.
