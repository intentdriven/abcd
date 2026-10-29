---
schema_version: 1
id: "iss-2610262031439210"
slug: "invariant-15-puts-the-session-transcript"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 168 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:167"
remedy: "In itd-59's scope, replace the spelled `history/<root_sha>` path with 'the native transcript store, reached through the history verb surface': adr-2610031751066232 spells the store as `transcripts/<root-sha>` under abcd's home and invariant 15 lets only the history package spell it."
related_intents: [itd-59]
finding_ref: cfk-62c2409c3599cc43
---

Invariant 15 puts the session-transcript store at `~/.abcd.noindex/transcripts/<root-sha>/`, but itd-59 feeds a corpus at `~/.abcd.noindex/history/<root_sha>/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:167` — "The session-transcript store (`~/.abcd.noindex/transcripts/<root-sha>/`"
- End B: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:36` — "lands in the native transcript corpus under `~/.abcd.noindex/history/<root_sha>/`"

Invariant 15 names the session-transcript store `~/.abcd.noindex/transcripts/<root-sha>/` (adr-2610031751066232) and says only the history package may spell the store's path, the legacy location included. itd-59 says every autonomous-run pass lands in the native transcript corpus under `~/.abcd.noindex/history/<root_sha>/`, and it keeps that store unchanged. It therefore targets a location the invariant says is not the store.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 168.
