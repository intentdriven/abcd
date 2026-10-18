---
schema_version: 1
id: "iss-2610171047098960"
slug: "abcd-implement-load-wait-renews-the"
severity: "minor"
category: "future-work-seed"
source: "agent-observation"
found_during: "building the load gate's renewal (iss-2610170926295770)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement_load.go"
remedy: "Keep the renewal running beside the gated command until it exits, under the same rules, if the coordinator rules that the run phase should be covered too"
refines: [iss-2610170926295770]
---

abcd implement load --wait renews the caller's reservations only while it waits: once the gate opens and the gated command runs, nothing renews them, so a make preflight that runs longer than half a two-hour lease can still let a reservation made by hand lapse before its push.

How close it is, as measured on 2026-10-17 (the coordinator left this captured, not built, on that day): a reservation's default lease is two hours, and the waiter renews one once it is an hour old, so the gated command starts with at least an hour of lease left, and more when a renewal came just before the open. That day's preflights on the development machine ran about 16 and 29 minutes once their gate opened, against waits of 11 minutes and 1 hour 23 minutes; the lapse that prompted iss-2610170926295770 fell inside the long wait, which the waiter now covers. A preflight would need to run for more than an hour after its gate opens before this gap lets a reservation lapse; a slower machine, a larger suite or a longer lease all move that line.
