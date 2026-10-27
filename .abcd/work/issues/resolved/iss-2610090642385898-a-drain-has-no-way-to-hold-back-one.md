---
schema_version: 1
id: "iss-2610090642385898"
slug: "a-drain-has-no-way-to-hold-back-one"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Add a skip-for-now to the drain: a list of issue ids or paths the current drain passes over, each with a reason, shown in the plan's passed entries and cleared when the drain ends."
resolution: "abcd drain --skip <iss-N|path> --reason <why> holds an issue, or every issue whose found_at lies in a path, back for the drain in progress; each move passes it over in passed with check skipped and the reason, --unskip lifts it, and the list clears when the drain ends."
impact: additive
resolved_by:
  commit: "fd748ff4a1cdc375a382696270181d7a57890263"
---

A drain has no way to hold back one issue or one package for now. A peer's verbal hold is invisible to it (only implement claim or a branch carrying the record keeps a lane off an issue), and while a fix is in flight in one package the only way to keep the drain out of that package is to stop the whole drain, so this run drove a colliding lane anyway and let whichever landed second rebase.

## Grounds

- pursued: a person can keep a drain off one issue or one package without stopping the drain; shown wrong if a held-back issue is still reserved, judged or given a lane, or if the list outlives the drain
