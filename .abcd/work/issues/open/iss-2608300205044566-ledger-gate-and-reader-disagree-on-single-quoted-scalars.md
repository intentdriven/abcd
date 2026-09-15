---
schema_version: 1
id: "iss-2608300205044566"
slug: "ledger-gate-and-reader-disagree-on-single-quoted-scalars"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 build, 2026-08-30"
found_at: "internal/core/lint/schema.go (issueScalar), internal/core/capture (decodeScalar)"
---

The committed-ledger gate and the ledger reader disagree on single-quoted scalars: issueScalar in the record-lint issue-shape check strips single quotes, capture's decodeScalar does not, so a hand-authored single-quoted value (severity: 'minor', lapsed_at: '2026-08-28T00:00:00Z') is lint-green while the reader refuses and skips the record invisibly. Pre-existing on unmodified code; every issue-shape check that reads a scalar inherits it, so the fix belongs in issueScalar and moves four checks at once.
