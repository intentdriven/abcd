---
schema_version: 1
id: "iss-2610262033155666"
slug: "itd-36-closed-as-delivered-has-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 381 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:479"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-6c91409dc87d2259
---

itd-36, closed as delivered, has the lifeboat licence gate refusing to surface kept originals; the launch chapter says that restrictive-licence gate is future and inert

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:479` — "curated project memory and provenance. At launch the gate is future and inert,"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:44` — "the lifeboat licence gate refuses to surface `sources/` files without an explicit allowlist entry"

The memory-unification intent, closed as delivered, says the lifeboat licence gate refuses to surface `sources/` files without an explicit allowlist entry, and says its provenance substrate provides the lifeboat's restrictive-licence gates. The launch chapter says the restrictive-licence gate over kept originals belongs to the lifeboat and is future and inert, and that no licence verdicts are rendered. Either the gate guards kept originals today or it has not been built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 381.
