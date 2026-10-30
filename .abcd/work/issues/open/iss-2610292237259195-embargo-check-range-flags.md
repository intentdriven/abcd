---
schema_version: 1
id: "iss-2610292237259195"
slug: "embargo-check-range-flags"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "lane E report relayed by the coordinator, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Judge only the commits the default branch's tip does not already reach, so a merge of the default branch brings in nothing to judge, and add a test with a branch that merged a default branch carrying an advisory-shaped id"
---

embargo check --range flags advisory-shaped ids that the default branch already carries once a branch has merged the default branch in, though the branch's own changes add none. The merge commit itself is not the cause: mergeAdds keeps only lines the merge adds against every parent. The likely cause is the range's commit list: rev-list a..b lists every commit reachable from b and not from a, so with a base older than the merged default branch, that branch's own commits brought in by the merge are judged as the branch's.
