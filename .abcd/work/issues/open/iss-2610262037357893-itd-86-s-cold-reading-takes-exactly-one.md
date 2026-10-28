---
schema_version: 1
id: "iss-2610262037357893"
slug: "itd-86-s-cold-reading-takes-exactly-one"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 734 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29"
remedy: "none (filed automatically)"
related_intents: [itd-86]
finding_ref: cfk-bfb7ab7a36b8e890
---

itd-86's cold reading takes exactly one target document; the implement brief says a cold reading is handed the union of every position's paths

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29` — "A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone."
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:154` — "The reading corpus is derived, never restated: the union of every position's `object.paths` in the checkout's committed `.abcd/config/reading-presets.json`"

itd-86 defines the cold reading as a pass over exactly one document, judged against the constraints visible in that document alone. The implement chapter derives what a cold reading is handed as the union of every position's `object.paths` in the reading presets, and an entry there may be a directory that covers everything beneath it. The two documents disagree on what the cold reading's input is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 734.
