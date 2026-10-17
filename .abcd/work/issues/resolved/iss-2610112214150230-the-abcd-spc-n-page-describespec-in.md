---
schema_version: 1
id: "iss-2610112214150230"
slug: "the-abcd-spc-n-page-describespec-in"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "lane for iss-2610111759346608, run-2610112143048230, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/record/record.go"
remedy: "Call intent.ReadyWithQuestions in describeSpec and in describeBundleSpec's per-member check, so a spec page withholds 'implement' from an intent with an open question (pointing at abcd intent ready, as its not-ready branch already does); test an open spec whose intent has one open question, and a bundle with one such member, watched fail first."
resolution: "describeSpec and describeBundleSpec now judge readiness with intent.ReadyWithQuestions, so a spec page withholds 'implement' from an intent (or bundle member) with an open question and defers to abcd intent ready, matching abcd itd-N, abcd intent ready and abcd build."
impact: fix
resolved_by:
  commit: "7c5347aaeba45cb603ef23fa92fe7710e9df3566"
---

The abcd spc-N page (describeSpec in internal/core/record/record.go) and a bundle's shared-spec page (describeBundleSpec) judge the spec's intent, or each live bundle member, with plain intent.Ready, so for an intent whose ## Open Questions still lists an unanswered item they say 'implement against this spec's body; when done, abcd spec close ...' while abcd itd-N, abcd intent ready and abcd build all say it is not ready. Confirmed for describeSpec by a scratch-repository run in the lane for iss-2610111759346608; describeBundleSpec by reading the code.

## Grounds

- pursued: abcd spc-N for an open spec whose intent (or any live bundle member) lists an unanswered open question says 'not ready' and names abcd intent ready for that intent; it would be shown wrong by such a page still saying 'implement against this spec's body' (pinned by TestDescribeSpecOpenQuestionsWithholdsImplement and TestDescribeBundleSpecOpenQuestionMemberWithholdsImplement).
