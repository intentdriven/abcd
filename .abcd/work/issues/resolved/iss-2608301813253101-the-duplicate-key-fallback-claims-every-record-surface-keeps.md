---
schema_version: 1
id: "iss-2608301813253101"
slug: "the-duplicate-key-fallback-claims-every-record-surface-keeps"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-delta-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "The duplicate-key fallback claims only what this rule's own scanner does — it keeps the first value, so a second line can silence a blocker armed on the value the first hides — which is what the rule's godoc licensed all along. Setting readerRefusesDuplicateKey on the dsp store was refused: that branch says the file is skipped by every disposition surface, and a duplicated-key disposition is read, found illegible, and reported as an unreadable answer. What each store's readers do with a duplicated key is established by exercising them, in TestDuplicateTopLevelKeyReaderByReader (internal/core/lint/duplicatekeyreaders_test.go) — read the table there rather than any prose. The nine-store enumeration this fix put in the declaration's godoc was itself overtaken: it gave each store ONE answer, and rdi and dsp each have a second reader that refuses the file (iss-2608301901264848). TestDuplicateKeyClaimIsScopedToThisRulesOwnScanner pins the message against the disposition reader's own answer, watched failing on the old wording."
impact: fix
resolved_by:
  intent: "itd-189"
---

the duplicate key fallback claims every record surface keeps the first value while the disposition reader discards both and reports the record illegible

Found by the itd-189 delta ruthless review. The message text is PRE-EXISTING;
the audit commissioned to bound it is BRANCH-INTRODUCED, and that is the part
worth recording.

The fallback branch asserts that the lenient scanner "every record surface reads
with" keeps the first value. That is false for the `dsp` store.
`issueschema.ParseDisposition` returns `DispositionRecord{ID: id}` on a
duplicated key, discarding BOTH values, under a comment that says so outright:
a duplicated top-level key is malformed to every reader of this ledger.

So one `abcd lint` run emits two findings on one file that contradict each
other. `record_schema` says the first value is kept, so the author concludes the
disposition still stands on `accepted`. `reading_outstanding` says no reader can
read the record and it carries no state. The second is the true one:
`standingDisposition` routes the item to `Unreadable` and its answer decides
nothing. `dsp` is armed in this repository's production record-lint config, so
this is reachable here and not only in principle.

**The recursion is the finding.** `1d06fe10` split `readerRefusesDuplicateKey`
out precisely because the previous claim had been carried across stores without
checking each. Its audit enumerated `adr`, `adm` and `srp`, and stopped one store
short of `dsp` -- committing the same error inside the fix for that error. Ninth
surface this cycle for the standing class, and the first that is recursive.

Remedy, and the reviewer is explicit that the obvious one is wrong. Do NOT set
`readerRefusesDuplicateKey: true` on `dsp`: that branch says the file is
"skipped by every disposition surface", and a duplicated-key disposition is not
skipped -- it is read, found illegible, and reported as an unreadable answer. It
would be a second wrong sentence.

Narrow the false universal to what the rule's own godoc already licenses: this
RULE'S scanner keeps only the first value, so a second line can silence a
blocker armed on the value the first hides. The godoc was already the narrower
and true form; the emitted message claimed more than its own documentation
licensed.
