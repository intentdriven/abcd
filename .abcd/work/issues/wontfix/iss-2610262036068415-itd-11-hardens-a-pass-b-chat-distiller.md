---
schema_version: 1
id: "iss-2610262036068415"
slug: "itd-11-hardens-a-pass-b-chat-distiller"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 624 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-11]
finding_ref: cfk-bc0b76e7688d9463
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032572271. Same premise in itd-11, here against the history chapter's statement that no shipped surface reads the transcript store but history itself."
duplicates: [iss-2610262032572271]
---

itd-11 hardens a Pass B chat-distiller that reads the transcript store; 11-history says no shipped surface reads the store but history itself.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:15` — "Pass B's chat-distiller reads the native redacted transcript store"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:393` — "No shipped surface reads the store but"

itd-11 is framed as hardening, not redesign, of a pipeline in which Pass B's chat-distiller reads the native redacted transcript store and has produced noisy findings. 11-history says the harvest out of the store is only a design: nothing does that yet, and no shipped surface reads the store but history itself. The hardening depends on a reader of the store that, as the history surface is written, does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 624.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032572271. Same premise in itd-11, here against the history chapter's statement that no shipped surface reads the transcript store but history itself.
