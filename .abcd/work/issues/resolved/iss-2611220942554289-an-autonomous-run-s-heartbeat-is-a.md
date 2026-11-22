---
schema_version: 1
id: "iss-2611220942554289"
slug: "an-autonomous-run-s-heartbeat-is-a"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/wake.go"
remedy: "Add one heartbeat verb (beside the autonomy wake) that does those checks deterministically: renew the run's reservations, report each tracked pull request as merged, queued with position, armed, or dropped with the reason, trim or warn at the disk floor, and print one line when nothing changed; the run's prompt then calls it and acts only on what it reports."
resolution: "abcd implement heartbeat renews the run's reservations, reports each tracked pull request and the free disk against the load gate's floor, and prints one line when nothing changed"
impact: additive
---

An autonomous run's heartbeat is a hand-written prompt that, every half hour, renews reservations, reads each tracked pull request, reads its merge-queue state, checks the free disk against the load gate's floor and reads the peers' notes. Over the night of 21-22 Nov 2026 it ran about thirty times, each a handful of shell commands composed by the agent, and each part could drift or be skipped.
