---
schema_version: 1
id: "iss-2611061902550488"
slug: "bare-drain-resumes-an-old-state-unsaid"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first real drain ahead of a long run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "Have a bare abcd drain open with one line saying whether it resumes a state, how old, and what it will do next, before it does anything; report a state with no end older than a few days in the dry run and the drive's consent screen, with the command that ends it; collapse the by-the-rule hand-backs in the summary to counts per reason."
---

A bare abcd drain resumed a drain state written fifteen days earlier, with no cap, without saying so at its start: it printed nothing for about twenty seconds, then, on an interrupt, a roughly 500-line summary that repeats every hand-back by the rule, and left a new host judgement waiting. Someone typing the bare verb to see what it does has no way to tell a read from a resume, and the drive's preconditions do not mention a stale state either.
