---
schema_version: 1
id: "iss-285"
slug: "quoted-yaml-nulls-split-capture-and"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "pr-294-review"
found_at: "internal/core/capture/parse.go"
resolution: "capture and record-lint now part bare from quoted YAML nulls identically. A bare null spelling normalises to the empty string at parse time while a quoted one stays the string it spells, so the nullness decision is made while the RAW scalar is still in hand — record-lint tests the raw scalar, and capture used to test the unquoted one. Widening IsNull for iss-287 made that split wider rather than narrower, which is why the two land together. An adversarial review found the empty spelling still split: impact: with no value took the nested-object branch and parsed to a map, which capture rejected as must be a string while record-lint read it as null. A lookahead now parts them — an object has an indented member, a bare key: does not."
impact: fix
---

Quoted YAML nulls split capture and record-lint verdicts: capture unquotes before IsNull, record-lint reads the raw value

An issue record carrying `impact: "NULL"`: capture's `decodeScalar`
(`internal/core/capture/parse.go:188`) strips the double quotes before
`validateStrict` calls `frontmatter.IsNull`, so `abcd capture resolve`
accepts the record — while record-lint's `checkIssueImpact` reads the raw
value with quotes intact, `ParseImpact` fails, and the `issue_impact_valid`
blocker fires on the record capture just accepted. The comment at
`internal/core/lint/lint.go:2397` claims this split is impossible. PR #294
widened the disagreement window from one spelling (`"null"`) to three; its
own tests pin both sides of the contradiction
(`internal/core/frontmatter/frontmatter_test.go:76` vs
`internal/core/lifeboat/graveyard_abandoned_test.go:167`). The fix is to
normalise quoting on one side, not to widen literal sets. Evidence:
`.abcd/work/reviews/2026-08-19-pr-294-null-predicate/` (F1, F2).
Forge mirror: #372
