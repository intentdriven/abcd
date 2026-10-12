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
resolution: "The contents part reads a heading's code spans once: headingText returns the masked text with its unmask, the anchor and the listed text read that, and two spans a stripped tag or link left touching are listed a space apart. TestContentsMasksCodeSpansOnce holds the record's headings."
impact: fix
resolved_by:
  commit: "fc0b8e5f472b08d39bbdc31baa1d806874e3efac"
---

The README contents part masks a heading's code spans twice, which is not idempotent. In readme_derive.go, headingText unmasks its result, then anchor (and escapeLinkText) re-run mdrecord.MaskCodeSpans on that output. When the first pass strips an inline tag or a link tail sitting directly between two code spans, the second pass pairs the spans across the join.

The first heading below is listed as the second line, where the forge anchors it #ab and shows two code spans; the third heading gets the right anchor but the merged text. The base before step 3 got these headings wrong too, so it is not a regression.

```markdown
## `a`<br>` b `
- [`a`` b `](#a-b-)
## `GET`<br>`POST`
```

## Grounds

- pursued: a heading with a tag or link between two code spans is anchored from both spans' rendered text and listed as two spans; the record's example heading, listed as one span or anchored other than #ab, would show it wrong.
