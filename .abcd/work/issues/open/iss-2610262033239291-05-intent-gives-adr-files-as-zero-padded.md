---
schema_version: 1
id: "iss-2610262033239291"
slug: "05-intent-gives-adr-files-as-zero-padded"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 399 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:93"
remedy: "none (filed automatically)"
finding_ref: cfk-534472ddc8b0a364
---

05-intent gives ADR files as zero-padded NNNN-< slug>.md, but 24-decide mints adr-< yymmddHHMMSS>< rrrr> ids with stamp-named files.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "(`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded)"
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:124` — "The id is `adr-<yymmddHHMMSS><rrrr>` — the twelve-digit UTC second stamp and the"

05-intent's itd-44 decision verdict sends capture to 'the existing ADR store' and gives its files as `NNNN-<slug>.md`, zero-padded. 24-decide says new ADR ids come from a timestamp mint that reads no maximum, with files named stamp then slug. Only 0001-0058 keep ordinal names, and a hand-numbered ordinal is exactly the collision the mint was adopted to close. A builder following 05-intent would write the retired filename shape.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 399.
