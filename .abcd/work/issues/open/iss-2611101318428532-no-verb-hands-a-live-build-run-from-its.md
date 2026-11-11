---
schema_version: 1
id: "iss-2611101318428532"
slug: "no-verb-hands-a-live-build-run-from-its"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/control.go"
remedy: "Add a handover the live driver runs, naming the receiving session, that moves the driver claim to it and has the reservation follow, so the receiver's take-over is accepted on the holder's own release; a test hands a live run over and steps it from the receiver."
---

No verb hands a live build run from its driver to another session, so the run state is edited by hand. implement leave refuses a session that never joined with --session, and implement resume refuses a take-over while the driver lives. On 8 Nov between 08:19 and 08:25, when a coordinating session ran out of context mid-run, the driving session removed its own entry from the run's control.json by hand, kept a backup copy and checked that the entry named its session and process, and the reservation was renewed by hand for the receiving session. iss-2610121528179959 covers a shared place for handover notes.
