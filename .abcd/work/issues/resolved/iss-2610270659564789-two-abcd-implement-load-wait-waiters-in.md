---
schema_version: 1
id: "iss-2610270659564789"
slug: "two-abcd-implement-load-wait-waiters-in"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "implement load --wait, two waiters in one worktree"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/renewdue.go"
remedy: "Hold one lock per copy around RenewDue's renew-and-note step, or re-read the reservation after a refused push and report a reservation a sibling waiter of this copy has just renewed as renewed, naming its new expiry, rather than as a failure."
resolution: "each RenewDue pass holds a per-copy lock, so a second waiter in the copy reads the tips the first has just pushed and finds nothing due instead of losing its pushes"
impact: fix
---

Two abcd implement load --wait waiters in one copy renew the same reservations at once, and the one that loses reports each loss as a fault. On 2026-10-27 at 01:51Z a preflight's waiter and a gated test run's waiter, both in one worktree, renewed the same five reservations together. One renewed all five. The other logged 'could not renew your reservation' for each, four of them as origin refusing a non-fast-forward reservation push, and the fifth as 'its latest commit is not one this copy pushed, so it is not renewed from here', though this copy had pushed that commit seconds before. Nothing lapsed, but the lines read as a reservation at risk and the last one names the wrong cause.
