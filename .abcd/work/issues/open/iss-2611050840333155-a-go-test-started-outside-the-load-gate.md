---
schema_version: 1
id: "iss-2611050840333155"
slug: "a-go-test-started-outside-the-load-gate"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "test runs started outside the load gate"
remedy: "Part of the fix is a product choice, to settle first: whether the runs are named in bare abcd status or on another surface, and whether the shell guard teaches the gated form in every managed repository when a session starts a go test outside abcd implement load, never refusing it. Then build what is chosen from the reader the gate already has (machineload.UngatedTests), with a test that starts an ungated run and asserts each chosen surface says so."
---

A go test started outside the load gate adds load that only a waiter can see. The gate names such runs by pid on its own waiting line (iss-2610241130157455) and the LOAD rules teach gating them, but nothing shows them when no waiter is there to print that line, and nothing tells a session or a sub-agent at the moment it starts one, so the one-minute load rises with no cause on record.
