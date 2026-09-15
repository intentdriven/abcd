---
schema_version: 1
id: "iss-2608301901264848"
slug: "the-nine-store-enumeration-checked-one-reader-per-store-so-r"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-fix-delta-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The nine-store enumeration is replaced by TestDuplicateTopLevelKeyReaderByReader, which establishes each reader's answer by exercising it through the entry point that reaches it in production: record.Describe, intent.Load, spec.Load, capture.List, capture.Disposition, capture.Promote, changelog.ShippedSince, lint.ReadReadingOutstanding and issueschema.ParseDisposition. A store whose readers disagree carries a row for each, which is the fact four rounds of prose kept losing: rdi and dsp each have a capture reader that REFUSES the file, and iss has a release-cut reader that keeps the first value. The declaration's godoc is now a pointer at the table. TestEveryStoreHasADuplicateKeyReaderRow refuses a store with no row, so a new store costs establishing its readers rather than being given an answer by analogy. Watched red first with the enumeration's own answers in the want column: four rows failed."
impact: internal
resolved_by:
  intent: "itd-189"
---

the nine store enumeration checked one reader per store so rdi and dsp are assigned an answer their second reader contradicts

Found by the itd-189 fix-delta ruthless review, which proved it by probe rather
than by reading, and verified independently by the orchestrator before capture.

The enumeration added in `b3e3d69c` walks nine STORES and gives each ONE answer.
Two stores have a second reader that contradicts the answer they were given.
`readingItemPosition` (reading.go, the `capture disposition rdi-N` path) and
`standingDispositionState` (promote.go) both call `parseFrontmatterAndBody`,
which refuses a duplicated key by name. So:

- `rdi` is listed as "keeps the first value". Its `capture` reader REFUSES.
- `dsp` is listed as "keeps neither". Its `capture` reader REFUSES, which is a
  different answer again.
- The `iss` row's "the one store the refusal branch is true of" is therefore
  false, read as a claim about readers.

The codebase says so itself, one line above the second reader:
`standingDispositionState`'s own comment opens "The third reader of a
disposition file" -- while the enumeration assigns that store one answer.

**This is the FOURTH turn on the duplicate-key claim family**, after
iss-2608301519254418, iss-2608301656200729 and iss-2608301813253101. Each fix
was correct and each was overtaken. The lineage is worth stating because it
names the design fault rather than the instance:

1. two message legs claimed a refusal that does not happen;
2. the flag justifying them was false for one store;
3. the fallback claimed a universal across stores;
4. the enumeration that replaced the universal checked one reader per store.

Each fix moved the claim one level more specific and none of them made it
CHECKABLE. That is the fault: an enumeration of another component's behaviour,
written as prose in a doc comment, cannot fail. It goes stale silently and is
believed because it is specific.

Remedy is therefore NOT a fifth prose correction. Make the enumeration
executable -- a test that probes each reader and asserts what it does -- and
reduce the comment to a pointer at it. A wrong row then fails a gate instead of
misleading a maintainer. The emitted messages need no change: the reviewer
confirmed they already say only what this rule's own scanner does, which is the
one claim that holds regardless of any reader.

The same wrong enumeration is copied into the `resolution:` field of
`iss-2608301813253101` in `resolved/` and must be corrected there too.
