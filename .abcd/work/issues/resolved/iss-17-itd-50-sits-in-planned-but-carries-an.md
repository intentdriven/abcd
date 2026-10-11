---
schema_version: 1
id: "iss-17"
slug: "itd-50-sits-in-planned-but-carries-an"
severity: "major"
impact: internal
category: "inconsistency"
source: "agent-finding"
found_during: "intent-dependency-sweep"
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md"
resolution: "Verified: nothing in this repo's Go tree implements the audit loop (no audit_mode / loop-to-acceptance / UNACHIEVABLE hits in internal/ or cmd/) — the implementation-complete table is the predecessor's spc-52 delivery, so itd-50 correctly stays in planned/ and no shipped/ move applies. The section is reframed as Prior Art (predecessor AC reconciliation, design input per the delivery-state provenance doctrine) so it no longer reads as a delivery claim in this repo."
---

itd-50 sits in planned/ but carries an implementation-complete AC reconciliation table; verify actual state and move through the shipped/ audit-notes path if true