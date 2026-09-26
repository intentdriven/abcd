---
schema_version: 1
id: "iss-2608301813259334"
slug: "three-nits-from-the-itd-189-delta-ruthless-review-including"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-delta-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "checkRecordJoins' godoc ordinals follow the code, iss-2608301744300631 names !!null, and commands/capture.md lists the two proposal join refusals."
impact: internal
resolved_by:
  commit: "6eed91a6"
---

three nits from the itd-189 delta ruthless review including a godoc whose ordinals no longer match the code order

Three nits from the itd-189 delta ruthless review, settled at the ship commit
rather than by re-opening a review.

1. `checkRecordJoins`'s godoc now reads first SPELLING, second PRESENCE, fourth
   POSITION, third BUCKET. The new paragraph was inserted above the one it
   follows in code, so the ordinals no longer match the order.
2. `iss-2608301744300631` enumerates `[]` and `{}` for the gate versus
   `describeADR` disagreement but omits `!!null`, which the same widening
   introduced and which behaves identically. The stated remedy already covers
   it; only the enumeration is short.
3. `commands/capture.md` lists the `grounds` and `proposal` refusals that hold
   today but not the two join refusals now armed on `proposal`: a target in
   another run, and a target outside the widening position.

## Grounds

- pursued: a reader following the godoc meets the legs in the order numbered and capture.md names every refusal record_schema arms on proposal; a refusal the page omits would show it wrong
