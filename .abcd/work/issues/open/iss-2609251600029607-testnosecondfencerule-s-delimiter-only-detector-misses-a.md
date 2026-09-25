---
schema_version: 1
id: "iss-2609251600029607"
slug: "testnosecondfencerule-s-delimiter-only-detector-misses-a"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrecord"
---

TestNoSecondFenceRule's delimiter-only detector misses a private toggle written as a literal regexp that matches a run (for example a pattern of 0-3 spaces then a backtick or tilde run, with a length check) or as byte comparisons on the first character; its doc says a pattern 'assembled at run time' escapes, which understates a literal regexp. Extend the detector to regexp literals whose pattern can match a fence run, and to first-byte comparisons against a backtick or tilde, or narrow the doc to the truth.
