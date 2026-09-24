---
schema_version: 1
id: "iss-2609240200426413"
slug: "abcd-iss-n-answers-that-the-record-is-not-found-in-the-issue"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A, itd4 verification review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/record/record.go"
---

`abcd <iss-N>` answers that the record is not found in the issue ledger when the record's file is there but was skipped on read, for example because it still carries a retired property such as `promoted_to`. describeIssue in internal/core/record/record.go reads only the parsed issues and never the skipped roster, so an adopter with an unmigrated ledger gets a false 'not found' with no hint to run `abcd capture migrate --apply`, while `abcd capture list` beside it names the remedy. The shape predates the related-links rename; the rename widens it to every unmigrated record.
