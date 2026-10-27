---
schema_version: 1
id: "iss-2610262040109467"
slug: "the-adapters-chapter-says-every-lifeboat"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 914 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:341"
remedy: "none (filed automatically)"
related_intents: [itd-95]
finding_ref: cfk-69843aeb4330f9a8
---

The adapters chapter says every lifeboat source reader implements probe → extract → copy; itd-95 says every adapter implements Section, Tier and Probe

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:341` — "of source readers, each implementing the `probe() → extract() → copy()` contract."
- End B: `.abcd/development/intents/shipped/itd-95-disembark-grounds-a-lifeboat-s-open.md:34` — "It implements the same `Source` interface every other adapter does (`Section()` / `Tier()` / `Probe()`)"

02-adapters says disembark reads a repository through source readers, each implementing the `probe() → extract() → copy()` contract. itd-95 adds an open-questions adapter that implements the same `Source` interface every other adapter does, which it gives as `Section()` / `Tier()` / `Probe()`. The two documents give different contracts for the same set of disembark source adapters.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 914.
