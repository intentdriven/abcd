---
schema_version: 1
id: "iss-2610262031258604"
slug: "04-tradeoffs-records-that-a-pack-either"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 127 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/03-evidence/04-tradeoffs.md:29"
remedy: "none (filed automatically)"
finding_ref: cfk-22b4bd78a97ac486
---

04-tradeoffs records that a pack either completes or writes nothing, with no state file at any path; the verification matrix stages a forensic checkpoint written on a mid-pack interrupt.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/03-evidence/04-tradeoffs.md:29` — "A pack is a single deterministic pass that either completes or writes nothing"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:40` — "A mid-pack interrupt writes a forensic checkpoint"

The tradeoff chose no checkpoint state: 'A pack is a single deterministic pass that either completes or writes nothing', and no run writes a _state.json at any path. The matrix's Disembark checkpoint row stages that 'A mid-pack interrupt writes a forensic checkpoint', for post-mortem only and at a location still undecided. A decided design target that writes state when a pack is interrupted contradicts the recorded choice that an incomplete pack writes nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 127.
