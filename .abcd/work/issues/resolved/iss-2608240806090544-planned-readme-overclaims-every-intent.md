---
schema_version: 1
id: "iss-2608240806090544"
slug: "planned-readme-overclaims-every-intent"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-a/round-5"
found_at: ".abcd/development/intents/planned/README.md"
resolution: "Rewrote planned/README.md subheading to the authoritative parent-README/contract wording; dropped the false categorical spec_id: spc-N claim (20/43 residents legally hold null). drafts/README refuter-cleared and left untouched."
impact: internal
resolved_by:
  commit: "52988d9"
---

planned/README.md categorically claims every intent carries a spec_id: spc-N link, but 20 of 43 planned intents have spec_id: null — a legal lint-enforced state (intent_lifecycle) per the surface contract, parent intents/README, and adr-34; the subheading silently reintroduces the over-narrow framing adr-34 already corrected in the parent README (resolved iss-3).