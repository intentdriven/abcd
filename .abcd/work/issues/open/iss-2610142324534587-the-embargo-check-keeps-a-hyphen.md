---
schema_version: 1
id: "iss-2610142324534587"
slug: "the-embargo-check-keeps-a-hyphen"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security-review-feat/embargo-steps-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Also match a variant of the text with each hyphen-plus-line-break removed."
---

The embargo check keeps a hyphen inserted at a line wrap: a title word split as over- and flow folds to over-flow, so a reflowed paragraph naming an embargoed title passes the outbound check unrefused.
