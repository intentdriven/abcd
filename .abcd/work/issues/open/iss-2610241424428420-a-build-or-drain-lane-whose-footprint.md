---
schema_version: 1
id: "iss-2610241424428420"
slug: "a-build-or-drain-lane-whose-footprint"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
remedy: "Give reserve (and build's backoff) a --wait form that blocks, with a timeout, until every overlapping reservation on origin is released or expired, fetching on an interval, and say in the exit-3 refusal which reservation blocks and that --wait exists."
---

A build or drain lane whose footprint overlaps a peer's live reservation backs off with exit 3, and nothing tells it when the reservation is released: on 2026-10-24 a planned build waited on a drain lane's reservation of internal/core/implement/loop, and the two sessions coordinated by cross-session messages, the drain promising to say when it released. A session with no peer messaging, or a hosted one, can only poll abcd peers.
