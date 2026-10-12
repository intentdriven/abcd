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
---

A README part's example can fail the question limits that R8 holds every tab to, because the lines are the project's own text quoted exactly. A heading such as '## Install — fast' lists as '- [Install — fast](#install--fast)', which question.CheckLimits refuses as an em dash in a list item, and a heading naming '__init__' is refused as bold markup. The cut fits only the rows rule, and the limits test's fixtures hold neither. Confirmed at the lane head after round 1's fixes with a README holding both headings: the contents tab raises em-dash-in-list-item and no-bold. Quoting the line differently would no longer be the exact line the part writes.
