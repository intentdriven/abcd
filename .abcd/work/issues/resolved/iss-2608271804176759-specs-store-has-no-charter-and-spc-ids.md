---
schema_version: 1
id: "iss-2608271804176759"
slug: "specs-store-has-no-charter-and-spc-ids"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/specs"
resolution: "specs store charter written (three READMEs: grammar, lifecycle, back-link checks, the live-vs-predecessor spc qualifier); the double-allocation citation cleanup remains tracked by iss-239"
impact: internal
resolved_by:
  commit: "8db72381"
---

the specs store has no charter and its id namespace is doubly allocated: specs/ is the only durable-tier record family with no README at any level (directory_coverage flags specs/, open/ and closed/ independently), the live store is spc-2..spc-42 while the retired predecessor store minted its own spc-1..spc-83, and above-ceiling references now collide — spc-33 and spc-37 resolve to live specs on different subjects than the retired records the citations meant. Write the three READMEs first (grammar, open-to-closed lifecycle, intent back-link, the no-status-field rule per adr-3, and the mandatory predecessor-store qualifier), then re-scope iss-239 with the corrected live range and the two collisions.