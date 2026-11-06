---
schema_version: 1
id: "iss-2611060810040824"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Add the stage's launch and refusals to the guard chapter's approver and fail-open sections, or point both at the drain chapter's walls."
---

`.abcd/development/brief/04-surfaces/17-guard.md:593-757` describes one bare launch, in which writes outside the working tree are judged by the protected set alone (`.abcd/development/brief/04-surfaces/17-guard.md:668`), and `commands/guard.md:424` says the same. A drive's stage launches walled, with hooks on and no `--bare` (`internal/core/runner/claude.go:190`), and the approver takes `--stage-home`, `--protect` and `--declared` (`internal/surface/cli/guard_approve.go:167`), which bound its writes and refuse a further agent. On a stage the hook denies a call it cannot judge (`internal/surface/cli/hook_occupancy.go:109`), against the chapter's fail-open account at `.abcd/development/brief/04-surfaces/17-guard.md:455-459`. `.abcd/development/brief/04-surfaces/35-drain.md:288-316` describes these walls; the guard chapter has only the three flags in its appendix, with no pointer to them.
