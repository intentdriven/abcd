---
schema_version: 1
id: "iss-2608261206508851"
slug: "the-intent-surface-contract-still-calls"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "bughunt-a/round-8"
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
resolution: "the intent surface contract drops the false 'shipped/ is empty' claim, aligning with the close-only entry criterion."
impact: internal
resolved_by:
  commit: "1ec3719f"
---

the intent surface contract still calls shipped empty while it holds intents