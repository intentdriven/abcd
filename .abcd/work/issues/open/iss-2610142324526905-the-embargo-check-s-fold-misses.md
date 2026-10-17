---
schema_version: 1
id: "iss-2610142324526905"
slug: "the-embargo-check-s-fold-misses"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security-review-feat/embargo-steps-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Drop every Default_Ignorable_Code_Point, map UTS #39 confusables for the GHSA and CVE alphabets, and decode HTML entities and backslash escapes before matching."
---

The embargo check's fold misses cross-script look-alikes, default-ignorable characters outside category Cf, and encoded forms: a Cyrillic or Greek letter standing in for a Latin one, U+034F, a variation selector or U+3164 inside an id or title, or an id written as HTML entities or with markdown backslash escapes passes the outbound check unrefused.
