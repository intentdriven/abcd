---
schema_version: 1
id: "iss-2610262034120625"
slug: "itd-26-builds-loot-s-refusing-licence"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 478 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-ce13d8316446edf5
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032585048. Same End A phrase and same root cause; loot's refusal is checked against its own acceptable-licences list (needs only the shipped licence detection), so the unbuilt lifeboat gate is not a dependency once the phrase is corrected."
duplicates: [iss-2610262032585048]
---

itd-26 builds loot's refusing licence check on itd-36's shared substrate, launch licence gate included; the library surface says that gate is not built and ingest never refuses on a licence

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "launch-gate-licence-gate machinery itd-26 needs is the **same** machinery memory ingest needs"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:362` — "is not built, and the shipped ingest classifies restrictive licences without"

itd-26 says the licence detection and launch licence gate machinery it needs is the same machinery memory ingest needs, and sits on top of it with no re-implementation, while loot refuses an incompatible licence. The library surface says the licence gate belongs to the lifeboat rather than launch, is not built, and that the shipped ingest classifies restrictive licences without ever refusing on them. The substrate as delivered cannot supply the refusal itd-26 depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 478.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032585048. Same End A phrase and same root cause; loot's refusal is checked against its own acceptable-licences list (needs only the shipped licence detection), so the unbuilt lifeboat gate is not a dependency once the phrase is corrected.
