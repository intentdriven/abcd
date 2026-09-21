---
schema_version: 1
id: "iss-12"
slug: "memory-shipped-vs-planned"
severity: "minor"
impact: internal
category: "inconsistency"
source: "review-followup"
found_during: "roadmap-consistency-review"
found_at: ".abcd/development/intents/shipped/itd-36-memory-unification.md"
resolution: "Memory docs no longer assert shipped state: spc-38/spc-39 references demoted from delivery claims to spec attribution, 07-memory header states itd-36 is planned/ and defers delivery state to the intent lifecycle, backed by the new brief-README provenance note."
---

memory shipped versus planned is ambiguous: docs say spc-38/spc-39 shipped, but itd-36 lives in planned and intents README says shipped is empty until Go capabilities ship.