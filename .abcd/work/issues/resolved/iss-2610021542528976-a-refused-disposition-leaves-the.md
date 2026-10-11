---
schema_version: 1
id: "iss-2610021542528976"
slug: "a-refused-disposition-leaves-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25: sec-postTagA"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reading.go"
remedy: "Move the redaction and prevalidateDisposition ahead of mutationPreamble in Disposition, after the read-only pre-flight readItemHead, as admit's requireWidening already sits; the writer re-runs the same check under the lock, so nothing it decides moves. Pin it with a test that removes the fixture's lock, sends each refused shape, and asserts no .iss-alloc.lock and no dispositions directory remain."
resolution: "Disposition redacts the request and runs prevalidateDisposition before mutationPreamble, so a disposition the schema refuses (no grounds, a hold with no exit condition, a state the position does not offer) of an item the ledger holds exits 2 and leaves no .iss-alloc.lock; TestRefusedDispositionOfAHeldItemLeavesNoLock, watched failing on all three shapes first."
impact: fix
resolved_by:
  commit: "b5fa626a7"
---

capture disposition refuses a malformed answer to an item the ledger holds only after mutationPreamble has run: Disposition (internal/core/capture/reading.go) redacts the request and calls prevalidateDisposition after the preamble, which opens the allocator lock file, so a disposition with empty grounds, a held state with no exit condition, or a state the item's position does not offer exits 2 but leaves .abcd/work/issues/.iss-alloc.lock in a ledger that had none. The file is gitignored and no directory is created, but the capture chapter promises a refusal writes nothing, and admit already checks its request (requireWidening) before the preamble. Found by the post-tag security review of 7fb52a6b5..d94472abb (sec-postTagA, LOW); unchanged from the range's base.

## Grounds

- pursued: every Disposition refusal raised from the caller's input writes nothing; a refused shape that still leaves the lock or a dispositions directory would show it wrong
