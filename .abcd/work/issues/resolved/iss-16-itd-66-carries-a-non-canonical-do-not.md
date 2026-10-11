---
schema_version: 1
id: "iss-16"
slug: "itd-66-carries-a-non-canonical-do-not"
severity: "major"
impact: internal
category: "inconsistency"
source: "agent-finding"
found_during: "intent-dependency-sweep"
found_at: ".abcd/development/intents/planned/itd-66-launch-payload-render-parity.md"
resolution: "Banner inverted the delivery-state provenance doctrine: spc-78 is predecessor attribution, not an authoritative contract for this rebuild. Do-not-implement banner removed; spc-78 and its two implementation deltas (config-file payload override, directory-convention smoke discovery) recorded as Prior Art + an Open Question for spec time; body stays canonical per the brief (04-surfaces/04-launch.md), so itd-66 is again implementable as itd-65's blocker."
---

itd-66 carries a NON-CANONICAL / do-not-implement banner while being itd-65's declared blocker; reconcile the intent body with the native spec that owns the render contract