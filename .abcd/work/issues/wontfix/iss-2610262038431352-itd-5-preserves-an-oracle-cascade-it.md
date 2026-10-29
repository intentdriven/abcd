---
schema_version: 1
id: "iss-2610262038431352"
slug: "itd-5-preserves-an-oracle-cascade-it"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 814 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:87"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-088804ba9251593e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031337587. Identical A end (itd-5 line 87) and the same no-fixed-cascade position, restated by 04-universal-patterns.md instead of 02-dependencies.md; one itd-5 edit resolves both."
duplicates: [iss-2610262031337587]
---

itd-5 preserves an oracle cascade it attributes to universal pattern § 7; 04-universal-patterns says the core imposes no cascade

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:87` — "Adding `capability_scope` does NOT modify the oracle cascade"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:60` — "Guidance the adapter layer offers, never a cascade the core imposes."

itd-5 says adding capability_scope leaves the oracle cascade unchanged. It cites § 7 as a 'fixed cascade' with an availability-driven fallback chain, and puts its selector above that cascade. 04-universal-patterns describes a native default that configuration overrides, and says the core imposes no cascade. The seam itd-5 promises to preserve is one the patterns page says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 814.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031337587. Identical A end (itd-5 line 87) and the same no-fixed-cascade position, restated by 04-universal-patterns.md instead of 02-dependencies.md; one itd-5 edit resolves both.
