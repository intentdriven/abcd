---
schema_version: 1
id: "iss-2610262040511595"
slug: "the-construal-glossary-describes-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 960 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/construal.md:20"
remedy: "The framing chapter (01-product/06-framing.md), its mapping row and tier hypothesis, and the first instance exist: ship or re-scope itd-143 to what remains (the interview landing)."
related_intents: [itd-143]
finding_ref: cfk-162b2a62e152cab5
---

The construal glossary describes the brief's framing chapter as existing with a construal; itd-143, still a draft, has that chapter and its first instance still to be written.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/construal.md:20` — "The brief's framing chapter opens with one, under a `## Construal` heading"
- End B: `.abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:55` — "This repository's own framing section, written as the first instance."

The stable construal entry says the brief's framing chapter opens with a construal under a ## Construal heading and quotes this repository's own. itd-143 is in drafts and its scope is to add the framing chapter under 01-product/ and write this repository's own framing section as the first instance. Either the chapter exists or that draft is still unbuilt; both cannot be true.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 960.
