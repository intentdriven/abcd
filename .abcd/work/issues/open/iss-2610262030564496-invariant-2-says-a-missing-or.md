---
schema_version: 1
id: "iss-2610262030564496"
slug: "invariant-2-says-a-missing-or"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 65 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-6bc27210b8f9e9f8
---

Invariant 2 says a missing or misbehaving adapter falls back to the native path and abcd never blocks; the reading chapter says a cold-reading dispatch whose provider cannot be reached exits 2 with nothing ingested.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:11` — "A missing or misbehaving adapter degrades to the native path; abcd never blocks on an absent tool."
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:377` — "a provider that could not be reached, a run id outside the run-id grammar, a run no assembly parked, and a dispatch that also names an output each exit 2 with nothing ingested."

03-invariants makes fallback non-negotiable: a missing or misbehaving adapter degrades to the native path, and abcd never blocks on an absent tool. 23-reading routes a position to a provider, which is an opt-in oracle adapter. When that provider could not be reached, the dispatch exits 2 with nothing ingested instead of falling back to the host-delegated harness, and only an explicit `host-decides` override keeps the run on the harness. An unreachable configured provider therefore blocks the reading instead of degrading.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 65.
