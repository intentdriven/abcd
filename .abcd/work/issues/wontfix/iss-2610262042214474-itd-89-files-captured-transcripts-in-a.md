---
schema_version: 1
id: "iss-2610262042214474"
slug: "itd-89-files-captured-transcripts-in-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1064 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-6ae91ea37aa7ef33
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same wording shift as the other itd-89 store records."
---

itd-89 files captured transcripts in a local per-repo store; the build sequence keeps the transcript store outside every checkout, at the user level.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "abcd redacts the transcript and files it in the local per-repo store"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:82` — "lives outside every checkout at the user level, so a repo tracks none of it"

itd-89 says the SessionEnd hook files the redacted transcript in the local per-repo store. The build sequence says the native transcript store lives outside every checkout at the user level, so no repo tracks any of it, and that pulling it into a checkout's local tier is a per-machine opt-in. The default home of the store differs between the two.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1064.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Same wording shift as the other itd-89 store records.
