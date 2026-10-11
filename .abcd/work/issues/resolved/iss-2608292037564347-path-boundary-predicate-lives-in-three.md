---
schema_version: 1
id: "iss-2608292037564347"
slug: "path-boundary-predicate-lives-in-three"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/fsutil/paths.go"
resolution: "fsutil exports IsPathSegmentByte and NameContinues as the one statement of the path boundary; RedactRoot's right boundary is the alphanumeric-only name rule and the scanner's isPathSegmentByte and nameContinues forward to fsutil, so the CLI error scrub, the install receipt and the store redactors agree on '<home>.', '<home>-old' and '<home>_x'. The leading half still differs by design for a multi-segment home under a longer root; captured separately."
impact: fix
resolved_by:
  commit: "5d8bdc084"
---

v0.6.9 combined ruthless review: three sites carry the same path-boundary predicate in different polarities and now disagree on a trailing suffix. fsutil.isPathBoundary (internal/fsutil/paths.go) and scanner.isPathSegmentByte (internal/adapter/scanner/identity.go) are the same byte class inverted, and scanner.nameContinues (internal/adapter/scanner/residual.go) is the home-path anchor's rule — alphanumeric only. So fsutil.RedactRoot / RedactHome leave 'cannot access /Users/<user>.' untouched (the '.' is a segment byte to them) while SweepCallerHome sweeps it, and the CLI error scrub and the install receipt disagree with the store redactors about the same sentence. Proposal: one home for the predicate — fsutil exports the boundary rule, scanner imports it, and the alnum-only trailing rule is the shared definition — taken in its own pass because fsutil's callers (error scrub, receipts, RedactRoot) are a different set from the scanner's.

## Grounds

- pursued: TestHomeRedactorsAgreeOnTheTrailingBoundary holds RedactRoot and SweepCallerHome to one answer for every printable byte after the home (RED on '.', '-', '_' before the change); a byte where they disagree, or a redaction of a longer alphanumeric name, would show it wrong
