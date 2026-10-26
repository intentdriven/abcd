---
schema_version: 1
id: "iss-2610262040135811"
slug: "the-configuration-chapter-keeps"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 917 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:463"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-017b441e7fa61d04
---

The configuration chapter keeps transcripts in a user-scope store under ~/.abcd.noindex/transcripts; itd-89 files them in a local per-repo store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:463` — "at `~/.abcd.noindex/transcripts/<root-sha>/`, holding redacted records and a staging area for raw transcripts awaiting redaction"
- End B: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "abcd redacts the transcript and files it in the local per-repo store"

03-configuration says the transcript corpus is a user-scope store at `~/.abcd.noindex/transcripts/<root-sha>/`, and that a checkout keeps its store in the local tier only when declared in local-transcript-roots. itd-89 says the SessionEnd hook files each transcript in the local per-repo store, with no flag. The default home of a captured transcript differs between the two.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 917.
