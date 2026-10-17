---
schema_version: 1
id: "iss-2610150551207288"
slug: "when-abcd-offers-the-product-thinker"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "planning interview of itd-2609151838312703, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question/asking.go"
remedy: "Make 'Unclear, explain with an example' a required option on every question to the product thinker: the asking rules in internal/core/question (asking.go and limits.go, and the GRILL domain and the commands pages generated from them), the question check, and the abcd:question-drafter agent all require it, counted within the four-option limit; choosing it re-asks the same decision rewritten as a story with one concrete example."
---

When abcd offers the product thinker options, there is no option for 'this question is unclear'. On 2026-10-15, at the planning interview for itd-2609151838312703, the product thinker could not follow a question about an overlapping draft and had to type 'explain, unclear' into the host's free-text field. The asking rules let an option set end only in 'Decide later' or 'None of these', so a person who does not understand the question has no option to say so. The product thinker's words: when offering options to the PT, always offer 'Unclear, explain with an example' to ensure the user can select that.
