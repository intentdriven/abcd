---
schema_version: 1
id: "iss-2610262035341739"
slug: "the-history-chapter-says-no-shipped"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 581 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:393"
remedy: "In 02-adapters.md:350 mark the transcript reader (and the memory reader row below it) as design targets, or remove them: internal/core/lifeboat imports neither internal/core/history nor the memory store, which matches 11-history.md:393."
finding_ref: cfk-a7b47d0893821ab1
---

The history chapter says no shipped surface but `history` reads the transcript store; the adapters chapter gives disembark a transcript reader that reads the same corpus.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:393` — "No shipped surface reads the store but `history` itself"
- End B: `.abcd/development/brief/05-internals/02-adapters.md:350` — "Reads the root-SHA-keyed local corpus; merge by timestamp/content hash when an imported specstory source is also present"

11-history says that nothing distils the store yet and that no shipped surface reads it except `history` itself. 02-adapters lists, among the source readers disembark uses to fill a lifeboat, a transcript reader over the native transcript store (`internal/core/history`). It describes that reader in the present tense and does not mark it as a design target, as it does for the `claude_md` reader's git history. Disembark is shipped, so either it reads the transcript store or the reader is unbuilt. The two chapters cannot both be right.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 581.
