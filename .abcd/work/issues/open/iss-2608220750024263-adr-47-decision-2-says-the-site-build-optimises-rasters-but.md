---
schema_version: 1
id: "iss-2608220750024263"
slug: "adr-47-decision-2-says-the-site-build-optimises-rasters-but"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-observation"
found_at: ".abcd/development/decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed C): adr-47 raster optimisation: sign off an image-codec dependency, commit pre-optimised assets, or amend adr-47?"
remedy: "Waits on ruling C (codec dependency, pre-optimised assets, or amend adr-47), noting a no-dependency path for PNG: if the build optimises, have internal/core/site re-encode each PNG with the standard library's png.Encoder at BestCompression and keep the smaller bytes, passing JPEG through verbatim, proven by a test that a fixture PNG shrinks and decodes to identical pixels; if assets are pre-optimised, add a site-render check that refuses a committed PNG the same re-encode shrinks by more than five per cent; if amended, strike 'optimises rasters' from adr-47 decision 2 by a superseding note."
---

adr-47 decision 2 says the site build optimises rasters, but the stdlib-only generator copies rasters verbatim; optimisation needs either an image-codec dependency (sign-off gate) or pre-optimised committed assets

## Remedy grounds (2026-09-29)

- image/png exposes CompressionLevel BestCompression (https://pkg.go.dev/image/png, consulted 2026-09-29). Measured on this tree's rasters on 2026-09-29: intro.png 1,096,607 to 904,437 bytes and logo.png 186,761 to 152,744 (about 18 per cent each); the two role portraits about 2 per cent. The re-encode drops ancillary chunks such as a colour profile, hence the identical-pixels test.
- Rejected: an image-codec dependency (WebP, AVIF), which needs a sign-off for a gain the standard library already captures in large part on this tree.
