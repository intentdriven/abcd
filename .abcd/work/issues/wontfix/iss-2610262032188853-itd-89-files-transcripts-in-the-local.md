---
schema_version: 1
id: "iss-2610262032188853"
slug: "itd-89-files-transcripts-in-the-local"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 249 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-7e70aaf766b85dd4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-89 itself says the user-level home had never been created and adr-29 keys the store on the root commit, so per-repo then meant keyed per repo; the later ADRs gave per-repo its in-checkout opt-in meaning."
---

itd-89 files transcripts in the local per-repo store; invariant 15 makes the user-level `~/.abcd.noindex/transcripts/<root-sha>/` the store, with a per-repo store only for a declared checkout

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "abcd redacts the transcript and files it in the local per-repo store"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:167` — "The session-transcript store (`~/.abcd.noindex/transcripts/<root-sha>/`, or `<repo>/.abcd/.work.local/transcripts/<root-sha>/` for a checkout declared in `~/.abcd.noindex/local-transcript-roots`"

itd-89's press release says abcd files each transcript in the local per-repo store. Invariant 15 names the session-transcript store as `~/.abcd.noindex/transcripts/<root-sha>/`, and uses `<repo>/.abcd/.work.local/transcripts/<root-sha>/` only for a checkout declared in `~/.abcd.noindex/local-transcript-roots`. One puts the default store in the user's home and the other in the repository.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 249.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-89 itself says the user-level home had never been created and adr-29 keys the store on the root commit, so per-repo then meant keyed per repo; the later ADRs gave per-repo its in-checkout opt-in meaning.
