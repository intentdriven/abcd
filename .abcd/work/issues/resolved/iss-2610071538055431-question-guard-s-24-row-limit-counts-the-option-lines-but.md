---
schema_version: 1
id: "iss-2610071538055431"
slug: "question-guard-s-24-row-limit-counts-the-option-lines-but"
severity: "minor"
category: "documentation"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071228211794 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd guard hook (question tool), GRILL asking rules"
remedy: "none (filed automatically)"
resolution: "the rows refusal names the whole tab and its split by part; the GRILL rule says the rows count the header, frame, question text and every option"
impact: fix
---

Question guard's 24-row limit counts the option lines, but the asking rule reads as a limit on the question text

The GRILL rule says one question, or one tab, fits twenty-four rows at eighty columns, and the refusal names "tab 1, question text (rows)". Asking a four-option role question, the guard refused 31 rows, then 25, then 25, while the question text itself was about 8 to 12 lines. The rows evidently include the option labels and descriptions, so the budget for material is far below 24, but neither the rule nor the refusal says so. Each refusal costs a rewrite by guesswork.

To see it: ask a question whose text is about 10 lines with four options whose descriptions are two sentences each; it is refused as over 24 rows of question text.

Remedy the reporter proposes: State the budget the check applies: either say the 24 rows include header, options and descriptions, or report the material rows left after the options in the refusal.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071228211794, a defect against abcd v0.13.1, surface abcd guard hook (question tool), GRILL asking rules.

Evidence:

- rpt-2610071228211794 (the report, kept in the inbox)
- lab-261002171217-c372ff6
