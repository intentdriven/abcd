---
schema_version: 1
id: "iss-2608220750024263"
slug: "adr-47-decision-2-says-the-site-build-optimises-rasters-but"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-observation"
found_at: ".abcd/development/decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): adr-47 raster optimisation: sign off an image-codec dependency, commit pre-optimised assets, or amend adr-47?"
---

adr-47 decision 2 says the site build optimises rasters, but the stdlib-only generator copies rasters verbatim; optimisation needs either an image-codec dependency (sign-off gate) or pre-optimised committed assets