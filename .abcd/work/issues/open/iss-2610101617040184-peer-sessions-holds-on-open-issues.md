---
schema_version: 1
id: "iss-2610101617040184"
slug: "peer-sessions-holds-on-open-issues"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "When the drain's skip-for-now (iss-2610090642385898) lands, let a session add a hold from outside the drain (a held_by entry naming the session, the reason and an expiry) that every drain and build <iss> reads, and make build <iss-N> --session take a claim from its start, as build <itd-N> --session already does."
---

Peer sessions' holds on open issues reached this drain only as chat: on 2026-10-10 three sessions (abcd-0b, abcd-a5, abcd-49) named eleven issues to keep out of an abcd drain, none of them through implement claim, because their lanes were build runs or plans that take no claim until a lane opens. The drain cannot see any of them, so the only guard was the driving session hand-skipping each one when a lane opened on it. Evidence for iss-2610090642385898, which a peer is building as the drain's skip-for-now; filed apart because that record is being edited on the peer's lane.
