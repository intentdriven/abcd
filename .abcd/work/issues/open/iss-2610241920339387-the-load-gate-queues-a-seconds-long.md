---
schema_version: 1
id: "iss-2610241920339387"
slug: "the-load-gate-queues-a-seconds-long"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd build itd-82 session, 2026-10-24, checking a two-package fix before re-queuing its preflight"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/queue.go"
remedy: "give the gate a size class for the command it runs, after a state-of-the-art check (Bazel's test size classes small, medium, large and enormous, each with its own timeout and resource share): a declared small command (a named package set, a short --timeout) is admitted beside the running work while the load is under its limit, without taking a preflight's slot, and only medium and larger commands queue for a slot"
---

The load gate queues a seconds-long command behind whole preflights: on 2026-10-24 'abcd implement load --wait --timeout 1h -- go test ./internal/core/spec/ ./internal/reachaudit/', two packages that run in about five seconds, waited the full hour behind four to five waiters and timed out without running, because the gate admits one command at a time in start order whatever its size. With the rule that day of no go test outside the gate, a session had no gated way to check a small change before re-queuing a full preflight, so it either skips the check or breaks the rule.
