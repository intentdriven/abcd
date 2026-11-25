---
schema_version: 1
id: "iss-2611251810162964"
slug: "a-machine-s-stuck-hand-back-bars-every"
severity: "minor"
category: "ux"
source: "plan-review"
found_during: "planning drives across machines, 2026-11-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_readmit.go"
remedy: "Using the count of hand-backs in a row for one reason that itd-2611141610197753's ninth criterion states (three with the same kind, stage and check, read from pace.stuck_refusals), mark a stuck hand-back made inside such a run as the machine's on origin; another copy's drain then takes the issue once as if it had not been handed back, and a second copy's stuck hand-back bars every copy as today. A test hands an issue back on one copy inside such a run and asserts a second copy takes it once."
related_intents: [itd-2611251809580662]
---

A lane handed back as stuck is marked on origin, and no drain takes the issue again until its record changes. When the cause is the machine and not the issue (an old build, a broken runner), one machine removes issues from every machine's queue: on 11 to 14 November nineteen lanes went this way for two causes already fixed on the default branch (.abcd/development/research/notes/drive-operations-2026-11.md, events 1 and 2). Split out of itd-2611251809580662 at its planning (its decision 12).
