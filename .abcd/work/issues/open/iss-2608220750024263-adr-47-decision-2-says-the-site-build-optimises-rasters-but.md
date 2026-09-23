---
schema_version: 1
id: "iss-2608220750024263"
slug: "adr-47-decision-2-says-the-site-build-optimises-rasters-but"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-observation"
found_at: ".abcd/development/decisions/adrs/0047-abcdev-app-rendered-from-this-repository-alone.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: adr-47 raster optimisation needs an image-codec dependency (sign-off), pre-optimised assets, or an ADR amendment). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

adr-47 decision 2 says the site build optimises rasters, but the stdlib-only generator copies rasters verbatim; optimisation needs either an image-codec dependency (sign-off gate) or pre-optimised committed assets