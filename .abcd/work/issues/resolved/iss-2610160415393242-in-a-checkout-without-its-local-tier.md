---
schema_version: 1
id: "iss-2610160415393242"
slug: "in-a-checkout-without-its-local-tier"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "messaging step 4 final review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/peers.go"
remedy: "Carry the unsighted state into peers' hold row and JSON (an unsighted field), and word the line as not counted, matching land's notice"
resolution: "abcd peers judges each release hold by the merge hold's own rule (claim.JudgedHolds): the cutting copy's own release reads as its cut, a release dated ahead with no sighting kept reads as not counted, and release_holds carries own and unsighted"
impact: fix
resolved_by:
  intent: "itd-2609151838312703"
  spec: "spc-2610150728396006"
---

In a checkout without its local tier, abcd peers lists a release reserved with a date more than five minutes ahead as holding merges, while abcd land and the implement land fence do not count it and print a notice; peers --json release_holds carries no field that says the hold was not counted.
