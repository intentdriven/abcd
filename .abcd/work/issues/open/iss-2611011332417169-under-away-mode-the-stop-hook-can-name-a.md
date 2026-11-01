---
schema_version: 1
id: "iss-2611011332417169"
slug: "under-away-mode-the-stop-hook-can-name-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "away mode, 2026-10-31 to 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/wake.go"
remedy: "Clamp the deadline wakeDeadline returns to no earlier than now plus a minimum lead (the host's cron lateness, a few minutes), so the named deadline is always one a wake scheduled now can meet; proven by a test where the owed set's WakeBy lies less than lease.Skew after now and the deadline comes out after now."
---

Under away mode the stop hook can name a wake deadline that is already past, and then no scheduled wake can release the stop. wakeDeadline (internal/core/autonomy/wake.go) takes the earlier of now plus the ceiling (the owed ceiling, or the idle ceiling when nothing is owed) and the owed set's WakeBy less lease.Skew. WakeBy comes from this copy's live reservations (owedReader.reservationItems: the renewal half-life while now is before it, else the effective expiry) and is never before now, but once now is within lease.Skew (five minutes) of it, WakeBy less the skew falls before now, so the hook names a deadline that has passed and that stays fixed for up to lease.Skew, until now reaches the half-life (and again before the expiry) or the reservation renews. On 2026-10-31 it named 18:28 GMT and kept naming it after 18:28, and on 2026-11-01 it did the same at 00:23 and 05:21; a cron set for the next minute was not counted, and the stop then fell to the unchanged-set row.
