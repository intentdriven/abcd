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
---

A drain has no way to hold back one issue or one package for now. A peer's verbal hold is invisible to it (only implement claim or a branch carrying the record keeps a lane off an issue), and while a fix is in flight in one package the only way to keep the drain out of that package is to stop the whole drain, so this run drove a colliding lane anyway and let whichever landed second rebase.
