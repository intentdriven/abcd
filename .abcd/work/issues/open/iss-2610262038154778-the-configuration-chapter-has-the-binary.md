---
schema_version: 1
id: "iss-2610262038154778"
slug: "the-configuration-chapter-has-the-binary"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 782 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:93"
remedy: "Edit the Rules-loader dedup row in 02-verification-matrix.md: say the primary refresh is event-driven but a fixed-N backstop (`rules.force_refresh_every_n`, default 15) does count prompts; drop 'Nothing counts prompts'."
finding_ref: cfk-fcab6019e0ea0b51
---

The configuration chapter has the binary read a prompt-counting refresh backstop; the verification matrix says nothing counts prompts.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:93` — "prompt-router refresh backstop (itd-3). The primary refresh is"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:73` — "Nothing counts prompts"

03-configuration lists `rules.force_refresh_every_n` (15) as a hand-set key the binary reads, a refresh backstop that re-injects always-relevant domains on a counter. The verification matrix's dedup row says the refresh is event-driven rather than turn-counted and nothing counts prompts. A shipped counter-based re-injection and no prompt counting at all cannot both be true.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 782.
