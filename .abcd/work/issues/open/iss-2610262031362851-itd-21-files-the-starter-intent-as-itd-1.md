---
schema_version: 1
id: "iss-2610262031362851"
slug: "itd-21-files-the-starter-intent-as-itd-1"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 151 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-21]
finding_ref: cfk-bed138686b91f0e8
---

itd-21 files the starter intent as itd-1, an id the collision-proof minting invariant does not allocate

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32` — "Starter intent (`/abcd:intent` interview kicked off as last step) capturing the project's mission as `itd-1`"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:159` — "the mint never allocates by looking at the current maximum"

itd-21 ends its scaffold with an interview capturing the project's mission as itd-1. Invariant 11 says record ids are collision-proof by construction and the mint never allocates by looking at the current maximum. A fixed or next-ordinal itd-1 is neither, and two minters would both claim it, so the draft depends on an id the allocator does not produce.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 151.
