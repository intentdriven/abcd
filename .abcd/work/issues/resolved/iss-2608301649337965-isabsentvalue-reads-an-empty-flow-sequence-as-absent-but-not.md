---
schema_version: 1
id: "iss-2608301649337965"
slug: "isabsentvalue-reads-an-empty-flow-sequence-as-absent-but-not"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-security"
found_at: "internal/core/lint/schema.go"
resolution: "isAbsentValue asks both flow collections from one place and knows YAML's explicit null tag, so the empty flow mapping and !!null are absences on the same terms the empty flow sequence already was: a hand-written admission carrying grounds: {} or grounds: !!null is a blocker finding rather than a groundless admission the report keys as admitted. The two spellings join blankSpellings in the capture parity test, and the combination counts the required-fields godoc, the parity test and iss-2608301308369559's resolution note state are corrected to the forty-two the enumeration now holds. The resolution note on iss-2608300935218982 is scoped to the spellings that change closed rather than claiming every blank. This change is scoped the same way: it closes three of the four spellings the body enumerates, and the fourth is not a predicate question. A trailing comment defeats every emptiness test at once, because the shared same-line scanner strips no comments and each test anchors on the last byte — so grounds: {}  # c, [] # c, ~ # c and "" # c all draw zero findings, and every same-line scalar in every store is read the same way. That is iss-2608301744268001's to close, in the scanner."
impact: fix
resolved_by:
  intent: "itd-189"
---

isAbsentValue reads an empty flow sequence as absent but not an empty flow mapping or an explicit null tag so a blank grounds passes the gate and silences the report

Found by the round-5 security review. Branch-introduced.

`isAbsentValue` special-cases the empty flow SEQUENCE and not the empty flow
MAPPING, and knows no explicit null tag:

```
if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") { ... }
```

So `grounds: {}`, `grounds: { }`, `grounds: {}  # c` and `grounds: !!null`
reach `checkRecordRequiredFields` as present and non-blank, draw zero findings,
and the admission is accepted. `admittedProposals` then keys the pair as
admitted and the proposal vanishes from the outstanding report: a groundless
admission answers a proposal and nothing says so.

What makes this an asymmetry rather than a missing feature is the siblings.
`[]`, `[ ]`, `""`, `''`, `" "`, `~`, `Null`, `|`, `|+` and `>-` are ALL caught.
One flow collection is handled and the other is not.

It also falsifies two claims this branch has already shipped. The resolution
note on iss-2608300935218982 says an empty, whitespace-only or quoted-whitespace
required field is refused in EVERY store that shares the check; and
`commands/capture.md` tells the user that `record_schema` refuses an admission
with a blank `grounds`. Both are currently false.
`capture/blankrequired_parity_test.go` enumerates exactly four blank spellings,
and neither of these is among them.

Remedy: one branch beside the flow-sequence one, plus the two spellings added to
`blankSpellings` and to the schema tests.
