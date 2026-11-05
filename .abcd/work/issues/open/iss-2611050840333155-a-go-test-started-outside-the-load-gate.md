---
schema_version: 1
id: "iss-2611050840333155"
slug: "a-go-test-started-outside-the-load-gate"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/race.go"
remedy: "Name the ungated go test runs in abcd's status output from the same reader (machineload.UngatedTests), and have the shell guard teach the gated form when a session starts a go test outside abcd implement load, without refusing it. A test starts an ungated run and asserts both say so."
---

A go test started outside the load gate adds load that only a waiter can see. The gate names such runs by pid on its own waiting line (iss-2610241130157455) and the LOAD rules teach gating them, but nothing shows them when no waiter is there to print that line, and nothing tells a session or a sub-agent at the moment it starts one, so the one-minute load rises with no cause on record.
