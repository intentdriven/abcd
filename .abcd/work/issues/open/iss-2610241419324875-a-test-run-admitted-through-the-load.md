---
schema_version: 1
id: "iss-2610241419324875"
slug: "a-test-run-admitted-through-the-load"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Have the gate notice a holder whose process tree has used almost no CPU for a set span while waiters queue, and say so to the waiters and in the holder's own output (naming its pid and checkout), so its owner can stop it by its own handle; never kill it for them"
---

A test run admitted through the load gate keeps the shared test slot for as long as its command runs, however little it does. On 2026-10-24 an agent's go test -timeout 60m of one package hung from 13:19 UTC at about 3% CPU; load-wait.json held the slot to 14:19 while about nine preflights from every session queued behind it, and it was freed only after a peer session noticed and asked its owner to stop it (45 minutes in). Nothing in the gate tells a slow holder from a stalled one, nor tells the holder's owner that others are waiting.
