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
---

TestNoSecondFenceRule (internal/core/mdrecord/fence_canonical_test.go) flags a file only when it writes a fence delimiter AND matches one of a few tracker spellings (inFence, inCode, openChar, fenceOpen, a boolean flipped in place as x = !x), so two common styles of a private fence toggle escape it: a regexp with run tracking kept in a variable named open, and a three-backtick flip written as an if/else rather than x = !x (mutants C and D on a scratch copy; A and B were caught). Only ten non-test Go files outside mdrecord write a delimiter at all, so the detector can be inverted to delimiter-only with a reasoned allowlist, and every new writer then needs a stated reason.
