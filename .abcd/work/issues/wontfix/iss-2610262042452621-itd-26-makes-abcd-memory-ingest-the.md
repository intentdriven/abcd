---
schema_version: 1
id: "iss-2610262042452621"
slug: "itd-26-makes-abcd-memory-ingest-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1091 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26, itd-2610091918433290]
finding_ref: cfk-a56f61bfb11f687c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031343432. Same End A line; the planned notes intent records the same merge into library."
duplicates: [iss-2610262031343432]
---

itd-26 makes /abcd:memory ingest the knowledge-ingest verb; itd-2610091918433290 makes /abcd:memory the agents' memory-notes command

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge)"
- End B: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:58` — "after the verb-split interview freed the name `memory` by merging the sources and memory commands into `library`; `/abcd:memory` is an agent's command."

itd-26 pairs its loot verb with /abcd:memory ingest for knowledge on a shared provenance substrate. itd-2610091918433290 records that memory was freed by folding the sources and memory commands into library, and that /abcd:memory is now an agent's command for notes. One verb name names two different capabilities.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1091.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031343432. Same End A line; the planned notes intent records the same merge into library.
