---
schema_version: 1
id: "iss-2610120129098206"
slug: "a-readme-part-s-example-can-fail-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "run-2610110908176735 lane-4 fix round 1 (spc-2610031704338417 step 4)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_ask.go"
remedy: "Hold a quoted example to the rows limit only: CheckLimits's style rules (no-bold, em-dash-in-list-item) judge the asker's own words, so exempt the lines a README part would write, or narrow the limits test's claim to rows for those lines and say so in the spec."
resolution: "Fixed by 1f89d667d (lane-4's fix round 1): both README cuts, a part's example and its reason for writing nothing, now refuse a candidate on any question.CheckLimits finding, not only the rows, so a quote stops before the first line that trips a style rule (an em dash in a list item, bold markup) and the tab stays within the limits; the limits test holds a README with both headings. The question pair had not been released, so no user met the defect."
impact: internal
resolved_by:
  spec: "spc-2610031704338417"
  commit: "1f89d667d057457705b8fdfcbdfabcf2d5bbe177"
---

A README part's example can fail the question limits that R8 holds every tab to, because the lines are the project's own text quoted exactly. A heading such as '## Install — fast' lists as '- [Install — fast](#install--fast)', which question.CheckLimits refuses as an em dash in a list item, and a heading naming '__init__' is refused as bold markup. The cut fits only the rows rule, and the limits test's fixtures hold neither. Confirmed at the lane head after round 1's fixes with a README holding both headings: the contents tab raises em-dash-in-list-item and no-bold. Quoting the line differently would no longer be the exact line the part writes.

## Grounds

- pursued: every README part's tab passes question.CheckLimits for any project's headings; a project heading that still makes a tab fail the limits would show it wrong
