---
schema_version: 1
id: "iss-2610112237411831"
slug: "the-readme-contents-part-masks-a-heading"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "run-2610110908176735 lane-3 review (spc-2610031704338417 step 3)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Mask a heading's code spans once and carry the spans through: headingText returns the masked text with its unmask, and anchor and escapeLinkText take that masked text instead of re-running mdrecord.MaskCodeSpans on unmasked output."
---

The README contents part masks a heading's code spans twice, which is not idempotent. In readme_derive.go, headingText unmasks its result, then anchor (and escapeLinkText) re-run mdrecord.MaskCodeSpans on that output. When the first pass strips an inline tag or a link tail sitting directly between two code spans, the second pass pairs the spans across the join. Example: the heading `## `a`<br>` b `` lists as `- [`a`` b `](#a-b-)`, where the forge anchors #ab and shows two spans; `## `GET`<br>`POST`` gets the right anchor but the merged text. The base before step 3 got that heading wrong too, so it is not a regression.
