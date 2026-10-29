---
schema_version: 1
id: "iss-2610262036186092"
slug: "the-decide-chapter-makes-abcd-decide"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 638 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/24-decide.md:16"
remedy: "In itd-44 (lines 120-126) route a confirmed decision verdict through abcd decide --adr instead of a second ADR-writing path in capture, so the decide mint stays the store's one minting verb; embark remains the restore writer."
related_intents: [itd-44]
finding_ref: cfk-f182f4fbc29c6252
---

The decide chapter makes abcd decide with the ADR flag the ADR store's minting verb, with embark counting only as a restore; itd-44 adds capture as a second path that writes new ADRs into the store.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/24-decide.md:16` — "**minting** verb, the counterpart of the read-only `abcd adr-N` dispatch"
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:120` — "A confirmed `decision` verdict DIVERTS capture"

24-decide names abcd decide with the ADR flag as the store's minting verb, the counterpart of the read-only adr dispatch, and accepts embark as another writer only because embark restores rather than mints. itd-44 has capture, on a confirmed decision verdict, write a new ADR into .abcd/development/decisions/adrs/. Two verbs would then mint ADRs, and the draft does not route through the decide mint.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 638.
