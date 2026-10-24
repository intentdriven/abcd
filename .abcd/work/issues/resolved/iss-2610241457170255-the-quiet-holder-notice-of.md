---
schema_version: 1
id: "iss-2610241457170255"
slug: "the-quiet-holder-notice-of"
severity: "major"
category: "ux"
source: "review-followup"
found_during: "review of capture/slot-stall"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_quiet.go"
remedy: "Have every waiting line show each holder ahead (hold record or waiter running its command) with its age and its tree's recent CPU since the waiter's first look, marked not yet measured until a span has passed, so busy, quiet and unmeasured holders can be told apart; share samples between waiters if the shared-file rule allows"
resolution: "Every waiting line now names each holder ahead after holding: (the hold record's holder and, from the process table, every waiter running its command or preflight in its test stages, or at the race gate every race run, of any account) with its age and its tree's CPU over the last ten minutes, or not yet measured until this waiter has watched it that long; the line is said again every ten minutes while the gate stays shut. Two limits remain and are documented: a child that starts and exits between looks is never seen, so a tree of short git children reads low and counts as working for the quiet notice; and only the own account's hold-record holder is said quiet with its checkout."
impact: fix
---

The quiet-holder notice of iss-2610241419324875 still leaves waiters unable to tell how a load-gate holder is doing in the case that prompted it. A slow holder whose tree keeps starting short git children counts as working and is never reported, which was the 45-minute run of 2026-10-24; silence is no positive sign that a holder is working, since a waiter that joined in the last ten minutes has not measured it yet; only a same-account hold record is watched, so a holder seen only in the process table (another account's waiter running its command) is never named; and each waiter needs ten minutes of its own samples before it can say anything.
