---
schema_version: 1
id: "iss-2610160608416611"
slug: "the-intro-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Change 'four further keys' to six and 'eleven' to thirteen at 03-configuration.md:6."
resolution: "the configuration brief's intro now counts seven hand-set keys and fourteen in all, matching the list once reservation.fork_origin joined it"
impact: internal
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

The intro at .abcd/development/brief/05-internals/03-configuration.md:6 says install writes four keys, --attribution one and --outbound two, and 'four further keys can be hand-set ... Beyond those eleven'. The same chapter counts 'Thirteen keys' at line 22 and 'the remaining six' hand-set at line 30: native_secret_scanning, rules.force_refresh_every_n, match.threshold, match.fields, land.push and land.enqueue.
