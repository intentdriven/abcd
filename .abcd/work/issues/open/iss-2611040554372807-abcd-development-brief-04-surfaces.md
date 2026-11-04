---
schema_version: 1
id: "iss-2611040554372807"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "List the pre-publication callers, or drop the count and the only-one claim."
---

.abcd/development/brief/04-surfaces/16-lint.md:87-92 says four surfaces judge the harness-leak class and that `abcd lint outbound` is the only one judging text before it is public. scanner.CheckOutbound also runs before text goes public at internal/core/guard/forge.go:523, internal/core/landing/prbody.go:115, internal/core/claim/prepare.go:426, internal/core/claim/withdrawforge.go:541 and internal/core/release/ingest.go:560, and internal/core/implement/loop/land.go:785 scrubs a pull request's text before posting.
