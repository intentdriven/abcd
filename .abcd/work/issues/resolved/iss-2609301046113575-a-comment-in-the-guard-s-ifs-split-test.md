---
schema_version: 1
id: "iss-2609301046113575"
slug: "a-comment-in-the-guard-s-ifs-split-test"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/guardset_test.go"
remedy: "Say what the example needs without the path: a macOS home under the Users directory whose account name holds a v, so the U and the v of an IFS of Uv split the home down to a bare root; abcd lint's privacy-hygiene rule (repolint rule_privacy.go, genericHomeRe) is the check that shows it gone."
resolution: "The comment names no absolute home path; abcd lint reports 0 errors at this commit."
impact: internal
resolved_by:
  commit: "718fd8243fbb60a818e02519aa64c43f323b0655"
---

A comment in the guard's IFS-split test (TestDefaultWordsSplitOnANamedIFSTheWrittenCompareReads, internal/core/guard/guardset_test.go) names a real macOS home directory, the developer account's own absolute path, as its example HOME, so abcd lint's privacy-hygiene rule fails the tree it lands in; it arrived with lane drainTrim and was found by integration 24b-2 before it reached main.

## Grounds

- pursued: abcd lint's privacy-hygiene rule passes on the tree; a committed file naming a /Users/<name> path again would show it wrong.
