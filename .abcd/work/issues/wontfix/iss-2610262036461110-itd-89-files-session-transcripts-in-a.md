---
schema_version: 1
id: "iss-2610262036461110"
slug: "itd-89-files-session-transcripts-in-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 673 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:338"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-9ac40bce975e9e91
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later accepted ADR names the user-level transcripts store and the local-transcript-roots opt-in, and internal/core/history/location.go implements it."
---

itd-89 files session transcripts in a local per-repo store; the history chapter says the store is user-level and the per-repo location is only an opt-in

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:338` — "The per-repo location is an opt-in pull."
- End B: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "files it in the local per-repo store"

itd-89's press release says the SessionEnd hook files the transcript 'in the local per-repo store', presenting per-repo as where every capture lands. The history chapter says the store is user-level, outside every repo, under ~/.abcd.noindex/transcripts/< root-sha>/records/, and that 'The per-repo location is an opt-in pull' honoured only for checkouts declared in ~/.abcd.noindex/local-transcript-roots. Both cannot describe the default destination of a captured transcript.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 673.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later accepted ADR names the user-level transcripts store and the local-transcript-roots opt-in, and internal/core/history/location.go implements it.
