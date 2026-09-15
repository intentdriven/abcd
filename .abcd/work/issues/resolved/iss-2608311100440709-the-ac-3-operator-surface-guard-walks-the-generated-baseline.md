---
schema_version: 1
id: "iss-2608311100440709"
slug: "the-ac-3-operator-surface-guard-walks-the-generated-baseline"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of the itd-184 fix commit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/regime_surface_test.go"
resolution: "the configuration walk skips the machine-written baselines by their -baseline.json shape, so cited URLs used as map keys can no longer turn ac-3 red"
impact: internal
resolved_by:
  intent: "itd-184"
  spec: "spc-62"
---

The ac-3 operator-surface guard walks the generated baseline caches under the abcd directory, whose map keys are arbitrary data: the citations baseline is machine-written from cited URLs, so a legitimate citation whose URL contains the substring 'regime' turns the guard red with findings that name no real defect. On a branch whose subject is supply regimes that is not a hypothetical, and the failure sends the reader hunting a regime knob that does not exist. A generated cache is not an operator surface and does not belong in the enumeration.

## Grounds

- pursued: a generated cache is not an operator surface, so excluding it removes a false red rather than a real channel; what would show this wrong is something coming to read a baseline as configuration, at which point the single stated exclusion is where that changes
