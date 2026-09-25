---
schema_version: 1
id: "iss-2609251510124162"
slug: "testnosecondfencerule-internal-core-mdrecord-fence-canonical"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrecord/fence_canonical_test.go"
resolution: "TestNoSecondFenceRule is delimiter-only: every non-test Go file outside mdrecord that writes a fence delimiter must appear in fenceWriters with a reason and a pinned occurrence count. The ten writers are named, most of them writers of output or one-line judges; a delimiter assembled at run time rather than written as a literal stays outside the detector's reach and is left to review."
impact: internal
resolved_by:
  commit: "fe1c9f5c"
---

TestNoSecondFenceRule (internal/core/mdrecord/fence_canonical_test.go) flags a file only when it writes a fence delimiter AND matches one of a few tracker spellings (inFence, inCode, openChar, fenceOpen, a boolean flipped in place as x = !x), so two common styles of a private fence toggle escape it: a regexp with run tracking kept in a variable named open, and a three-backtick flip written as an if/else rather than x = !x (mutants C and D on a scratch copy; A and B were caught). Only ten non-test Go files outside mdrecord write a delimiter at all, so the detector can be inverted to delimiter-only with a reasoned allowlist, and every new writer then needs a stated reason.

## Grounds

- pursued: on a scratch copy, mutant C (a regexp with run tracking in a variable named open, in a new file) and mutant D (an if/else flip on three backticks appended to an allowlisted file) each fail the detector, and it passes on the tree; a toggle written with a literal delimiter that the detector passes would show it wrong
