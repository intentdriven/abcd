---
schema_version: 1
id: "iss-2608280813441672"
slug: "the-rules-loader-documentation-domain"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "itd-141-planning-interview"
found_at: ".abcd/rules.json"
resolution: "rules-loader DOCUMENTATION text updated to the adr-54 reality: em-dash rule enforced as a banned token, casing checks review by nature"
impact: fix
---

the rules-loader DOCUMENTATION domain text in .abcd/rules.json still claims the punctuation lint is staged as itd-141, which adr-54 falsified at the planning interview