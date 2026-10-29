---
schema_version: 1
id: "iss-2610262039086156"
slug: "itd-59-puts-the-transcript-corpus-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 841 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-59]
finding_ref: cfk-c28eb6d0b645d3e2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033019185. Same draft edit as iss-2610262033019185, set against a different statement of the same current state."
duplicates: [iss-2610262033019185]
---

itd-59 puts the transcript corpus at `~/.abcd.noindex/history/<root_sha>/`, but the configuration page puts it in the sibling store `~/.abcd.noindex/transcripts/<root-sha>/`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19` — "an operator's interactive session is redirected into the native transcript corpus at `~/.abcd.noindex/history/<root_sha>/`"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:463` — "at `~/.abcd.noindex/transcripts/<root-sha>/`, holding redacted records and a"

itd-59 exists to feed autonomous-run transcripts into the corpus it places under `history/`, and it keeps that path as-is. The configuration page says the history store holds the repository registry, and that the transcript corpus is a sibling store at `transcripts/<root-sha>/`, not part of the registry's tree. The two documents name different directories for the same corpus, so a build from itd-59 would write where no transcript resolver reads.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 841.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033019185. Same draft edit as iss-2610262033019185, set against a different statement of the same current state.
