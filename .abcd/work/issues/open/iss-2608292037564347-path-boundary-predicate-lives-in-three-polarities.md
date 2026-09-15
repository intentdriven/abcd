---
schema_version: 1
id: "iss-2608292037564347"
slug: "path-boundary-predicate-lives-in-three-polarities"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/fsutil/paths.go"
---

v0.6.9 combined ruthless review: three sites carry the same path-boundary predicate in different polarities and now disagree on a trailing suffix. fsutil.isPathBoundary (internal/fsutil/paths.go) and scanner.isPathSegmentByte (internal/adapter/scanner/identity.go) are the same byte class inverted, and scanner.nameContinues (internal/adapter/scanner/residual.go) is the home-path anchor's rule — alphanumeric only. So fsutil.RedactRoot / RedactHome leave 'cannot access /Users/<user>.' untouched (the '.' is a segment byte to them) while SweepCallerHome sweeps it, and the CLI error scrub and the install receipt disagree with the store redactors about the same sentence. Proposal: one home for the predicate — fsutil exports the boundary rule, scanner imports it, and the alnum-only trailing rule is the shared definition — taken in its own pass because fsutil's callers (error scrub, receipts, RedactRoot) are a different set from the scanner's.
