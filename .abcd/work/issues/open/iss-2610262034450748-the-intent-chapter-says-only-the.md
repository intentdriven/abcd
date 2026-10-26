---
schema_version: 1
id: "iss-2610262034450748"
slug: "the-intent-chapter-says-only-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 520 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:366"
remedy: "none (filed automatically)"
related_intents: [itd-2609020625400169]
finding_ref: cfk-0db6cbf423e03b12
---

The intent chapter says only the reading-ingest verb writes origin contributed-by-reading; itd-2609020625400169 has the capture promote path stamp it on the draft it mints.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:366` — "contributed-by-reading < rdg-N>/< rdi-N> (the reading-ingest verb only)"
- End B: `.abcd/development/intents/shipped/itd-2609020625400169-an-intent-that-a-reading-occasioned-says.md:33` — "stamps `origin: contributed-by-reading <rdg-N>/<rdi-N>` on the draft it mints"

05-intent's frontmatter template annotates the origin value contributed-by-reading < rdg-N>/< rdi-N> as written by the reading-ingest verb only. itd-2609020625400169 says the promote path for an accepted reading item stamps that value on the intent draft it mints, and that the stamp primitive accepts the value only from a caller supplying the run and item pair. The two documents name different verbs as the sole writer of the same intent field.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 520.
