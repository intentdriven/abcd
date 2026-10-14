---
schema_version: 1
id: "iss-2610140638074587"
slug: "the-load-waiters-hold-is-per-account"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd-22 preflight load-slot lane, 2026-10-14"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Gate without a file another account can write: read the machine's process table, which shows every account's processes, and stay shut while any other make preflight, an earlier or command-running abcd implement load --wait, or a go test -race run (each judged from its arguments, never a self-matching pattern) is live under any account; after seeing the machine free wait a random 15-60s and re-check everything before opening; keep load-wait.json as an extra within one account, and run make preflight's test stages through this gate."
---

The load waiter's one-waiter-at-a-time hold, ~/.abcd.noindex/load-wait.json, lives in the caller's own home, so it is per account, not per machine as the help and the LOAD rule call it. A second macOS account on the same machine (a colleague joining this machine) runs its own waiter against its own home and never sees the first account's hold, so two accounts' gated preflights are admitted together whenever the load and race-run conditions are open at once, which is exactly the moment between a preflight's start and its race step. Sharing the hold file across accounts is not a fix: a file another account can write would let one account hold every other account's gate shut, which the security rule forbids.
