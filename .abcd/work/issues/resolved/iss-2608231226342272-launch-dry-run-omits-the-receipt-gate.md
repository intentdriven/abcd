---
schema_version: 1
id: "iss-2608231226342272"
slug: "launch-dry-run-omits-the-receipt-gate"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "v0.6.2 release failure post-mortem 2026-08-23"
found_at: "internal/core/launch/dryrun.go"
resolution: "abcd launch --dry-run now carries a semantic-receipts row in every state, reporting which receipts are recorded for the candidate commit and pointing at the runbook. It reports presence only and never a pass, so release.yml stays the sole trust root for the required-gates list."
impact: fix
resolved_by:
  commit: "1ff105b"
---

`abcd launch --dry-run` reports a `gates` array built in
`internal/core/launch/dryrun.go`: `secret+pii-scan`, `marker-block`,
`installability-smoke`, `documentation-auditor`, `citation-baseline`.
`receipt_gate` is not among them, and no other field reports it.

So the preview whose stated purpose is "the bundle, the scan, and the gates"
answers cleanly while the gate that will actually refuse the release goes
unmentioned. Run before the v0.6.2 cut, it reported version 0.6.2, 105 bundle
files, zero secret/PII hard-fails, smoke ok, 51 citations with receipts — and
said nothing about the missing semantic receipts that then failed the release.

This is a `loud-staging` violation of the subtler kind. The two
`not_implemented` gates in that list are exemplary: they announce that they did
not run. `receipt_gate` is worse off than unimplemented — it is fully
implemented, it is armed at release time, and the preview simply does not know
it exists. An absent row reads as "no such gate", which is the one thing it is
not.

The preview must not become a second trust root: `release.yml` deliberately owns
the required-gates list, because the workflow rather than the committer-editable
in-tree config is what decides the gate's teeth. So the preview should report
what receipts are RECORDED for the candidate commit and point at the runbook,
not assert which gates are required.