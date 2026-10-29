---
schema_version: 1
id: "iss-2610262040403877"
slug: "itd-26-calls-the-substrate-s-knowledge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 947 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-2482df61c4dc0703
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031343432. Same End A line; End B is the correct name the draft should adopt."
duplicates: [iss-2610262031343432]
---

itd-26 calls the substrate's knowledge consumer /abcd:memory ingest; the provenance substrate calls it /abcd:library ingest.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge)"
- End B: `.abcd/development/brief/05-internals/09-provenance-substrate.md:3` — "A separable plumbing layer used by both `/abcd:library ingest` (per itd-36) and `/abcd:loot`"

itd-26 says the shared provenance substrate has two consumers, /abcd:loot for code and /abcd:memory ingest for knowledge. 09-provenance-substrate names the knowledge consumer /abcd:library ingest (itd-36). That is two verb names for one consumer.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 947.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031343432. Same End A line; End B is the correct name the draft should adopt.
