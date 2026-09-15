---
schema_version: 1
id: "iss-2608301901260678"
slug: "the-refusal-tells-an-issue-author-the-record-is-skipped-by-e"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-fix-delta-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The refusal names the ledger reader and says what does NOT skip the record: 'the issue ledger reader refuses a duplicated key, so capture skips the file - the release cut does not: it reads the same record on its first value, as this rule's own scanner does'. The old clause said the file was skipped by every issue surface, while changelog.ShippedSince -> newRecord reads the same resolved record with frontmatter.Fields, first-wins, and folds its title, summary, shipped_in and impact into the release cut. Both halves are rows in TestDuplicateTopLevelKeyReaderByReader, established by running the readers rather than asserted about them, and TestTheIssueRefusalNamesTheLedgerReaderNotEverySurface pins the message against the release-cut probe - watched failing on the old wording."
impact: fix
resolved_by:
  intent: "itd-189"
---

the refusal tells an issue author the record is skipped by every issue surface while the changelog derivation reads resolved records leniently

Found by the itd-189 fix-delta ruthless review. PRE-EXISTING text; the delta
rewrote this line's tail and re-blessed the clause, which is why it is captured
now.

The refusal branch tells an issue author the file "is skipped by every issue
surface". `capture`'s `scanLedger` does route it to `Skipped` -- but the
changelog derivation does not. `changelog/shipped.go`'s `newRecord` reads the
same file with `frontmatter.Fields`, first-wins, and folds its title, summary,
`shipped_in` and `impact` into the release cut.

So an author told the record is invisible everywhere finds it in the generated
CHANGELOG section. Same class as the record this delta was closing, one surface
further out, and it is the reason the remedy on iss-2608301901264848 is to make
such claims executable rather than to keep narrowing them by hand.
