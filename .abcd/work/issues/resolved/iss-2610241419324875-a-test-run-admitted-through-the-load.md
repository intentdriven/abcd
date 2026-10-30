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
remedy: "Have the gate notice a holder whose whole process tree has used little CPU for a set span while waiters queue, and say so as a fact, not a diagnosis, to the waiters and in the holder's own output (naming its pid, checkout and how long it has held the slot), so its owner can judge it and stop it by its own pid if it is stuck; never kill it for them"
resolution: "The load gate now follows the CPU of a command hold's whole process tree between looks (a child gone between looks counts as work, and the reader's own ps is left out). Once the tree has used no more than a tenth of a core for ten minutes, each waiter says so as a fact in a further waiting line, naming the holder's pid, checkout and how long it has held the slot, and the holder says the same on its own stderr while others queue at any gate it holds. It is no diagnosis and nothing acts on it: the hold stays and nothing is signalled. It does not cover the case that prompted it: a slow tree that keeps starting short git children counts as working and is never reported, so silence is no sign that a holder is working; only same-account hold records are watched; and each waiter needs ten minutes of its own samples. That remainder is iss-2610241457170255."
impact: fix
---

A test run admitted through the load gate keeps the shared test slot for as long as its command runs. On 2026-10-24 an agent's go test -timeout 60m of the loop suite, started at 13:19 UTC, held the slot through load-wait.json for 45 minutes while about nine preflights from every session queued behind it. It was a long, slow run, not a hung one: its goroutine dump showed it still working through tests at 30 to 70 seconds each under load, mostly in git subprocesses, so its test binary showed little CPU. The waiters could not tell whether it was working; a peer session noticed and asked its owner, who stopped it. Nothing in the gate tells the waiters how the holder is doing, nor tells the holder's owner that others are waiting.
