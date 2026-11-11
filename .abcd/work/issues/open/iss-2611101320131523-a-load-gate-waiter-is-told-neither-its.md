---
schema_version: 1
id: "iss-2611101320131523"
slug: "a-load-gate-waiter-is-told-neither-its"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Waits on a ruling first: then show each waiter its position and an estimate from past run times, drop a waiter whose holding process has ended, and read the race slot count from the person's own load settings."
---

A load-gate waiter is told neither its place in the queue nor when it may start, nothing notices a waiter whose session has ended, and race runs go one at a time per account. On 7 Nov at 12:58 five branches on one account had waited 2 to 4 hours while load sat near 10 against a limit of 32, and mapping waiters to their worktrees took a hand lsof pass; at 11:12 a pull request had waited three hours for a slot after a 40-minute run of its own; on 8 Nov at 08:46 seven preflights were queued at load 48. iss-2610230426248185 covers a waiter's misleading wait lines. A person must choose how many race runs an account may run at once, and whether the person can raise that number.
