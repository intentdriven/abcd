---
schema_version: 1
id: "iss-2608311207501002"
slug: "the-carriers-floor-added-to-the-read-block-eval-proves-a-car"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of the itd-186 fix commit"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_oracle_test.go"
resolution: "Each carrier now declares a cold marker string drawn from its own travelling text, and requireCarriers demands that marker in the bundle's raw bytes as well as the path in the manifest. Emitting an empty Text for every bundle item now fails, naming the carrier and the assertions it disarmed."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

The carriers floor added to the read-block eval proves a carrier's PATH is in the manifest and never that its BYTES are in the bundle, so a carrier can arrive empty and every absence assertion over it is vacuous. Emitting empty text for the projected candidates leaves requireCarriers satisfied while the WARM-FIELD and DRAFT-ORIGIN assertions and the item-text half of the field-absence check assert nothing at all; stripping the whole-file text of the disciplines and specs carriers is green the same way. Four of the five carrier rows are affected, and only the two the negative control happens to cover are pinned at content level. The floor was added in the previous round to close exactly this class -- an absence assertion with nothing underneath it -- and it was placed one dimension short of the thing it was meant to measure.

## Grounds

- pursued: a manifest names what an assembly says it passed and only the bundle says what it actually passed, so a content oracle has to read the content. Watched red against the empty-text mutant. This is wrong if a marker is ever chosen from text the redaction removes, which would make the floor fire on a correct assembler.
