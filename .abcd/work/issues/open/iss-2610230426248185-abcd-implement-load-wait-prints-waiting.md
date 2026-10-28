---
schema_version: 1
id: "iss-2610230426248185"
slug: "abcd-implement-load-wait-prints-waiting"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "lane D preflight queue, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Print a line each time the set of conditions holding the gate shut changes (including after a settle whose re-check finds the gate shut again), naming what now holds it, so the waiter's log always reflects its current state; keep it to one line per change."
---

abcd implement load --wait prints 'waiting for capacity: ...' once, at the first look that finds the gate shut (OnWait), and 'load gate: the machine looks free; checking everything again in Ns before opening' once (OnSettle); a waiter whose re-check finds a new blocker goes back to waiting silently. On 2026-10-23 a waiter's log read 'the machine looks free' for over half an hour while two other waiters' preflights ran ahead of it, so the log, the only thing a session reads while it waits, said the opposite of what was happening.
