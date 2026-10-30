---
schema_version: 1
id: "iss-2610222251373104"
slug: "a-plain-make-preflight-on-a-shared"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "drain lane run-2610222011116629, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Give make preflight's own load gate a bound no shorter than the wrapper form's eight hours (or read it from the person's load configuration), so a plain preflight never times out of a queue it is still moving through, and say in commands/implement.md what a timed-out preflight should do."
---

A plain make preflight on a shared machine can wait at the load gate indefinitely: its gate is bounded at two hours (Makefile, implement load --wait --timeout 2h), and when other accounts' preflights queue ahead with eight-hour waiters, each about an hour long, it times out and its retry joins the back of the queue, behind every waiter that arrived meanwhile. A drain lane's implementer timed out once and was re-queued behind three more on 2026-10-22. commands/implement.md says to run make preflight plain, while the wrapper form (implement load --wait --timeout 8h -- make preflight) keeps a queue place without the two-hour bound.
