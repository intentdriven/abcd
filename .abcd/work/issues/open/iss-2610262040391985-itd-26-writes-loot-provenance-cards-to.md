---
schema_version: 1
id: "iss-2610262040391985"
slug: "itd-26-writes-loot-provenance-cards-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 946 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:44"
remedy: "Edit 09-provenance-substrate line 109: write the loot citation path as `.abcd/work/loot/<source>.md`, as itd-26 does throughout, in place of `.abcd/development/loot/<source>.md`."
related_intents: [itd-26]
finding_ref: cfk-2f1c50e7b242d176
---

itd-26 writes loot provenance cards to .abcd/work/loot/; the provenance substrate puts loot citations in .abcd/development/loot/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:44` — "write provenance card to `.abcd/work/loot/<source>.md`"
- End B: `.abcd/development/brief/05-internals/09-provenance-substrate.md:109` — "citation generation in `.abcd/development/loot/<source>.md`"

itd-26 writes each loot provenance card, which carries the citation and licence, to .abcd/work/loot/< source>.md in the working tier. 09-provenance-substrate says loot's citation generation writes to .abcd/development/loot/< source>.md in the durable tier. The one card cannot live in both tiers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 946.
