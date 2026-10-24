---
schema_version: 1
id: "iss-2610241214065079"
slug: "the-load-gate-meters-only-a-preflight-s"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24, seven sessions preflighting on one machine"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Put every heavy preflight stage behind the load gate, not only the test stages: the lint gates, the smoke and cold-reading eval lanes, and build and vet wait at a gate of their own (or the same one) with its own slot count in load, so N sessions preflighting at once run at most the configured stages concurrently, and the gate reports how long each stage waited."
---

The load gate meters only a preflight's test stages; every stage before them runs ungated. On 2026-10-24 seven sessions on one 10-core machine each ran make preflight, and all seven ran their lint gates, the smoke and cold-reading eval lanes (five to six minutes each) and build and vet at once, holding the load average near 25. Tests timed out under that load (internal/surface/cli passed go test's ten-minute default), and every preflight then queued behind one test slot, so one nitpick fix waited over an hour. Raising load.preflights to 2 widens only the test slot; nothing bounds the stages before it.
