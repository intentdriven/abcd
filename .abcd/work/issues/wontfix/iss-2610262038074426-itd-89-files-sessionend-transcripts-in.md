---
schema_version: 1
id: "iss-2610262038074426"
slug: "itd-89-files-sessionend-transcripts-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 773 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-297105ea8a1f4b3e
refines: [iss-2608230817034768]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-89's per-repo store reads as a store keyed per repository, which is what the later ADRs and 31-lab describe."
---

itd-89 files SessionEnd transcripts in the local per-repo store; 31-lab says the transcript store is machine-scoped and keyed on the root commit

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "files it in the local per-repo store"
- End B: `.abcd/development/brief/04-surfaces/31-lab.md:30` — "The lab store is machine-scoped and keyed on the repository's root commit, as the transcript store is"

itd-89 states the SessionEnd hook files each session's transcript in the local per-repo store. 31-lab states the lab store is machine-scoped and keyed on the repository's root commit, as the transcript store is, so the transcript store is a machine-level store keyed per repository rather than a store local to the repository. The two place the captured transcript in different stores.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 773.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-89's per-repo store reads as a store keyed per repository, which is what the later ADRs and 31-lab describe.
