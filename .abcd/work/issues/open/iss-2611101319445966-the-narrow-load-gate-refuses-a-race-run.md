---
schema_version: 1
id: "iss-2611101319445966"
slug: "the-narrow-load-gate-refuses-a-race-run"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/narrow.go"
remedy: "Give the narrow gate a race route for a few named packages, kept apart from other race stages as the race gate requires, and let it split a long package into several test processes as preflight's sharding does; a test admits a narrow race run of one package while a whole run waits."
---

The narrow load gate refuses a race run and cannot split a long package, so a lane cannot get a race pass, or a full pass of a long package, without queuing behind whole preflights. The narrow rule (internal/core/machineload/narrow.go) reads any -race run as a whole run, and the implement loop package cannot finish on the narrow gate's two cores inside 40 minutes. On 7 Nov a lane's race pass sat more than 46 minutes behind other preflights; at 09:30 the loop package hit its limit twice, 40 minutes narrow and 25 under race; at 19:22 one step of a lane had spent about 7 hours at the gate. c27f0f793 changed the lane brief to run race tests as a gated whole run of their own; the gate itself is unchanged at this branch's head.
