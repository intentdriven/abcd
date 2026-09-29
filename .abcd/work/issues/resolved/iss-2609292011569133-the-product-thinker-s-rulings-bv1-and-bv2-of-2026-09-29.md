---
schema_version: 1
id: "iss-2609292011569133"
slug: "the-product-thinker-s-rulings-bv1-and-bv2-of-2026-09-29"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusblock/statusblock.go"
remedy: "Build BV1 and BV2 in statusblock.Read and the text board, note the change on itd-2609212103568351's Audit Notes, and supersede adr-2609212115255771 with an ADR whose decision 2 excludes an intent in a lane (ruling H10 of 2026-09-29)."
resolution: "Built BV1 and BV2 in statusblock.Read and the text board; itd-2609212103568351 carries an Audit Notes line; adr-2609292012006845 supersedes adr-2609212115255771 with decision 2 revised (ruling H10)."
impact: additive
resolved_by:
  commit: "3263b2c91"
---

The product thinker's rulings BV1 and BV2 of 2026-09-29 change what the shipped status block promises: the text board gives Later as a count alone (--json and the site keep its rows), and an intent in a lane is listed under Now only, never also under Next or Later. That makes itd-2609212103568351 criteria 1 and 3 and adr-2609212115255771 decision 2 (Next is every planned intent the gate reports READY; Later is the rest) false as written.

## Grounds

- pursued: we expect an intent in a lane to be listed under Now only on the board, --json and the site, and Later to be a count on the text board; shown wrong if TestAnIntentInALaneIsOnlyUnderNow or TestBoardRendersLaterAsACount fails, or a board shows a lane's intent twice
