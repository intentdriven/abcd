---
schema_version: 1
id: "iss-2611040712366155"
slug: "abcd-library-ledger-with-no-flag-exits-2"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/source.go"
remedy: "Name --list in the refusal, or have the bare form with no input list; a test runs it on an empty input."
---

abcd library ledger with no flag exits 2 on a real corpus with a message about an invalid record, because the bare form appends, and it does not point at --list; a published advisory tells people to run that bare form to review flips.
