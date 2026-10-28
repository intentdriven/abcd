---
schema_version: 1
id: "iss-2610262033019185"
slug: "itd-59-puts-the-transcript-corpus-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 348 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-59]
finding_ref: cfk-988c0d136c6f70ed
---

itd-59 puts the transcript corpus at ~/.abcd.noindex/history/< root_sha>/; the ahoy page says history/ is the registry only and holds no transcripts.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:36` — "every autonomous-run pass's instructions + reasoning lands in the native transcript corpus under `~/.abcd.noindex/history/<root_sha>/`"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:427` — "the REGISTRY only: identity and lineage keyed on the root-commit SHA. ahoy owns it; it holds no transcripts"

itd-59's whole scope is landing autonomous-run transcripts in the native transcript corpus under ~/.abcd.noindex/history/< root_sha>/, and it treats that store as fixed. The ahoy page's inventory says history/ is the registry only, owned by ahoy, and holds no transcripts, and it puts the corpus in a sibling directory, transcripts/< root-sha>/. The two cannot both be right about where the corpus lives. Built from itd-59 as written, capture would write into the registry directory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 348.
