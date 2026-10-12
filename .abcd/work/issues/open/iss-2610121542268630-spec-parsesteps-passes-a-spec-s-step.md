---
schema_version: 1
id: "iss-2610121542268630"
slug: "spec-parsesteps-passes-a-spec-s-step"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec"
remedy: "Mask or refuse control and bidi runes in a step title where ParseSteps reads it (a spec is the owner's own text, so masking keeps legitimate titles), with tests for the parse and for the loop's echoes."
---

spec.ParseSteps passes a spec's step titles through raw, and the build loop stores and echoes them (brief, state.json, implement status --json, pull-request titles), so a spec whose step title carries a control or bidi rune puts it on those surfaces.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
