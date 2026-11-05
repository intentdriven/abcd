---
schema_version: 1
id: "iss-2611050136321099"
slug: "a-narrow-run-reads-a-whole-waiter-s-age"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/narrow.go"
remedy: "Record when a slot was first seen free for the whole-run queue, in a file an account's narrow waiters share, and judge overdue from that time; a wait's own looks cannot do it, since a fresh narrow run under a steady stream has seen nothing yet, and another account's slots stay unread, so measure whether a minute's stand-aside costs enough to justify it first"
---

A narrow run reads a whole waiter's age as how long it has been overdue, not how long a slot has been free for it, so a waiter that queued for forty minutes is overdue the moment a preflight exits, and narrow runs stand aside for the minute or so it takes to open; a bounded cost, which a short stand-aside no longer spends from a wait's five minutes
