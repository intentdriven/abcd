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
---

The tool installer carries a second CI detector (internal/core/tools/install.go reads CI != "") beside the canonical one in internal/core/implement/load.go (ciRunner: GITHUB_ACTIONS=true, or CI other than empty, false or 0); the copy misses a runner that sets only GITHUB_ACTIONS, and two detectors drift. One primitive should serve both, with the installer keeping its stricter refusal of any non-empty CI.
