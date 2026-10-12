---
schema_version: 1
id: "iss-2610112036089151"
slug: "the-findingguardfailed-doc-comment-in"
severity: "nitpick"
category: "drift"
source: "impl-review"
found_during: "lane for iss-2610101803326081, run-2610111930237804, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/findings.go"
remedy: "Rewrite the FindingGuardFailed doc comment to name all three lists GuardFindings fails on (Unfixed: an open blocking finding; Deleted: a blocking record in open/ at the anchor that is gone without a resolution; Uncommitted: an open record that differs from HEAD, at any severity), matching the guard's code and the launch brief's refusal-kinds row; comment-only, checked by reading GuardFindings."
---

The FindingGuardFailed doc comment in internal/core/changelog/findings.go says the guard fails when 'a consequential finding this cycle produced is still open and unwaived', but GuardFindings also fails on an open issue record that differs from HEAD (at any severity) and on deleted records, so the comment states one case of three. A sibling of iss-2610101803326081, whose lane fixed the launch brief's table row and RefusalUnfixedFinding's comment in release/emit.go and left this second package for capture.
