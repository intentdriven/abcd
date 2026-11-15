---
schema_version: 1
id: "iss-2611150554145414"
slug: "a-load-gate-waiter-renews-every"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/renewdue.go"
remedy: "renew from the waiter only reservations whose run the loop reads as still needing them, or skip a reservation whose record logs a failed release"
---

A load-gate waiter renews every reservation the copy last pushed, including one whose lane was handed back and whose release failed, so that reservation does not lapse while the copy runs gated tests
