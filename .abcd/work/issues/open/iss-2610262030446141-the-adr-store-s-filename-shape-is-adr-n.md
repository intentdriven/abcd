---
schema_version: 1
id: "iss-2610262030446141"
slug: "the-adr-store-s-filename-shape-is-adr-n"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 39 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:74"
remedy: "Edit 03-mental-model.md and 05-intent.md to name the ADR store's files by the id without its adr- prefix: `<stamp>-<slug>.md` for ids minted since adr-45 and `NNNN-<slug>.md` for the ordinals 0001-0058."
finding_ref: cfk-2a2480b4074403cc
---

The ADR store's filename shape is adr-N-< slug>.md in the mental model and zero-padded NNNN-< slug>.md in the intent chapter

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "routes capture to the existing **ADR store** (`.abcd/development/decisions/adrs/`, `adr-N-<slug>.md`)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "the existing ADR store (`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded)"

Both passages describe where a confirmed decision verdict is written, the same ADR store, but 03-mental-model names its files adr-N-< slug>.md and 05-intent names them NNNN-< slug>.md, zero-padded. A writer following either produces a filename the other does not describe.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 39.
