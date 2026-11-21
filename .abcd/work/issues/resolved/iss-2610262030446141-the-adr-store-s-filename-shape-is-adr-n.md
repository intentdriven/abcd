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
resolution: "Both ends already name the ADR store's files by the id without its adr- prefix: 03-mental-model.md since 68be032a1 (`<stamp>-<slug>.md`, ordinals 0001-0058 keeping `NNNN-<slug>.md`) and 05-intent.md since f8de4f61c (the stamp spelled out as `<yymmddHHMMSS><rrrr>-<slug>.md`, 0001-0058 keeping `NNNN-<slug>.md`). This commit adds a test in internal/core/decide that holds both passages to the stamp form and the ordinal form, and refuses each passage as it stood before its fix."
impact: internal
resolved_by:
  commit: "3f56ff10eb9f0093f72bd82868b23fac5447036a"
---

The ADR store's filename shape is adr-N-< slug>.md in the mental model and zero-padded NNNN-< slug>.md in the intent chapter

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "routes capture to the existing **ADR store** (`.abcd/development/decisions/adrs/`, `adr-N-<slug>.md`)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "the existing ADR store (`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded)"

Both passages describe where a confirmed decision verdict is written, the same ADR store, but 03-mental-model names its files adr-N-< slug>.md and 05-intent names them NNNN-< slug>.md, zero-padded. A writer following either produces a filename the other does not describe.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 39.

## Grounds

- pursued: the two fourth-capture-verdict passages describe the same filenames, so a writer following either produces a name the other describes; TestBriefNamesADRFilesTheSameWayAtBothEnds would show it wrong by failing if either passage loses the stamp form or the ordinal form, or brings back an adr-N- prefix.
