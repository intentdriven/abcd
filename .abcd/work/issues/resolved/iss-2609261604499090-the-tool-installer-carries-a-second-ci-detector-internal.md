---
schema_version: 1
id: "iss-2609261604499090"
slug: "the-tool-installer-carries-a-second-ci-detector-internal"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
resolution: "internal/cienv.Runner is the one CI detector: the load check and the tool installer both call it and core/implement's copy is gone; the installer also refuses any non-empty CI by an explicit, commented extra check, so GITHUB_ACTIONS=true alone and CI=false both refuse an install."
impact: fix
resolved_by:
  commit: "7f34670d"
---

The tool installer carries a second CI detector (internal/core/tools/install.go reads CI != "") beside the canonical one in internal/core/implement/load.go (ciRunner: GITHUB_ACTIONS=true, or CI other than empty, false or 0); the copy misses a runner that sets only GITHUB_ACTIONS, and two detectors drift. One primitive should serve both, with the installer keeping its stricter refusal of any non-empty CI.

## Grounds

- pursued: one primitive answers is-this-CI for every verb that asks; a second detector reappearing, or an install proceeding with GITHUB_ACTIONS=true or any CI value set, would show it wrong
