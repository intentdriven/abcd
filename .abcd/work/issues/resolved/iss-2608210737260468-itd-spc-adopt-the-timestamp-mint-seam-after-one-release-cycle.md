---
schema_version: 1
id: "iss-2608210737260468"
slug: "itd-spc-adopt-the-timestamp-mint-seam-after-one-release-cycle"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-114 ship review"
resolution: "Intents and specs mint through the shared recordid timestamp seam (adr-45 ruling 3): the max+1 refs-union scans, the mint warnings and the integer-ceiling guards are deleted, and the per-checkout locks now only serialise the presence check and the write inside one checkout."
impact: fix
resolved_by:
  commit: "ff0fab7fd50d4c460899fdfb159243f8d78c55f3"
---

The intent (itd) and spec (spc) families still mint via the legacy max+1 refs-union allocator; adr-45 ruling 3 schedules their adoption of the timestamp-numeric mint seam as configuration — a swap of the allocation call in the intent and spec stores onto the shared recordid seam, retiring their MaxAcrossRefs scans and mint warnings — after the captures family has run the scheme for a release cycle (itd-114 ac-6's Given clause). This is the tracker so the rollout's second family is not forgotten once that cycle completes

## Grounds

- pursued: two current checkouts minting itd/spc in the same window allocate distinct ids by construction once the mint reads the clock and entropy instead of a maximum; a same-id pair across two checkouts after this commit would show it wrong
