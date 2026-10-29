---
schema_version: 1
id: "iss-2610262033503137"
slug: "the-memory-substrate-says-a-failed"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 441 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:140"
remedy: "Edit 07-memory.md section 5 (coverage check): drop 'refuses next quotation block'; say a failed cumulative check is an MQ002 blocker that makes `abcd library lint` exit nonzero, and nothing refuses at ingest."
finding_ref: cfk-791a99c89edbe335
---

The memory substrate says a failed coverage check refuses the next quotation block from that source; the library chapter says nothing enforces the budget at ingest.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:140` — "On fail: refuses next quotation block from that source until coverage drops via summarisation rewrite of existing pages."
- End B: `.abcd/development/brief/04-surfaces/07-library.md:405` — "computed at lint time. Nothing enforces it at ingest."

07-memory says that when the cumulative coverage check fails, abcd 'refuses next quotation block from that source until coverage drops'. 07-library says the budget is computed at lint time and 'Nothing enforces it at ingest', and that the lint mutates no store state. With ingest unenforced and the lint write-free, nothing is left to make the refusal 07-memory describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 441.
