---
schema_version: 1
id: "iss-2608301803426383"
slug: "three-doc-comments-outlived-the-tests-and-the-rule-they-desc"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-delta-ruthless"
found_at: "internal/core/lint/schema_test.go"
resolution: "the three orphaned doc comments are deleted: two in schema_test.go naming tests that no longer exist and claiming a reader-mirroring the gate deliberately does not do, and one in schema_parity_test.go contradicting the live test eight lines below it"
impact: internal
resolved_by:
  intent: "itd-179"
---

three doc comments outlived the tests and the rule they describe and one contradicts the comment eight lines below it

Found by the itd-179 delta review. Branch-introduced, and it is the delta's own
declared defect class: three doc comments outlived the tests and the rule they
describe.

`schema_test.go` carries a comment saying the committed-ledger gate refuses
exactly what capture's reader refuses, and that the closed vocabulary is read
from one copy so the two can never disagree. Both halves are now false:
`checkIssueRecordShape` no longer parses the value, and capture's reader
tolerates every spelling. A second names a test that was deleted.

The sharpest is in `schema_parity_test.go`, where an orphaned block says the
gate must reach the same answer as the reader, while the live test's own comment
eight lines below says the gate deliberately does NOT mirror the reader and that
this is the point of the rule. One file, two opposite claims, eight lines apart.

Left by the delta's own deletions rather than written wrong, which is the usual
way this class arrives.

## Grounds

- pursued: we expect a comment describing a deleted test to mislead a reader more than its absence would, so the deletion is the correction
