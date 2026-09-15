---
schema_version: 1
id: "iss-2608311207560513"
slug: "four-of-the-read-block-eval-s-oracle-tables-carry-no-emptine"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "delta review of the itd-186 fix commit"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_oracle_test.go"
resolution: "requireOracleTables asserts the declared count of all eight oracle tables and runs from materialise, so every corpus test carries it. The counts are duplicated by hand rather than derived from len(), because a derived count agrees with the table by construction."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

Four of the read-block eval's oracle tables carry no emptiness floor, so silently emptying any of them is green: carriers, excludedKeys, excludedHeadings and excludedFamilies. Emptying carriers and re-breaking the assembler's store node type restores exactly the defect the previous round was opened to close. Emptying excludedKeys with excludedHeadings disarms the field-absence check entirely, and emptying excludedFamilies disarms half the family-absence check. The holes table did get a floor in that round, but it is a greater-than-zero test over a two-row table, so it can still halve without noticing. A floor of greater-than-zero on a table whose size is known is not a floor; each table should assert its declared count.

## Grounds

- pursued: a greater-than-zero floor on a table whose size is known is not a floor. Watched red against an emptied carriers table combined with a broken record-graph enumeration, the pairing that restored the previous round's defect. This is wrong if the counts are updated reflexively to make a failing run pass, which turns the guard into a formality.
