---
schema_version: 1
id: "iss-2610102142432879"
slug: "a-peer-s-announced-record-freeze-cannot-hold-pull-requests"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/implement.md"
remedy: "Give a record freeze a form the merge path reads: a session announcing one runs a verb that lists every open PR armed for auto-merge that touches .abcd/ records, disarms them with a note naming the freeze, and re-arms them when the freeze lifts; the implement loop's landing checks for a live freeze before arming."
---

A peer's announced record freeze cannot hold pull requests that were armed for auto-merge before it began. On 2026-10-10 a session opened a freeze so it could rename every record with a long slug (#904), and the drain session parked its new lanes. But a drain lane PR (#901), armed by the implement loop's landing an hour earlier, merged through the queue during the freeze. Its record move under the old long name broke #904's queued tree, which dropped #904 from the queue and cost about an hour of re-merging and re-queueing. Nothing listed the PRs each session had armed, and nothing let the freeze disarm them.
