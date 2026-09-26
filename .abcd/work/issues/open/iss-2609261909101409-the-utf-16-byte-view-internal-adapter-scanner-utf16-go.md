---
schema_version: 1
id: "iss-2609261909101409"
slug: "the-utf-16-byte-view-internal-adapter-scanner-utf16-go"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/utf16.go"
---

The UTF-16 byte view (internal/adapter/scanner/utf16.go, utf16TextRune) ends a run at a surrogate, so a character outside the Basic Multilingual Plane, an emoji written as a surrogate pair, cuts a byte-order-marked UTF-16 string in two, and a name after it is never read when fewer than two code units stood before the pair: an emoji, a space and a name behind a byte-order mark yields no view at all, while the same name before the emoji is found. Detector: the caller's name after an emoji in one byte-order-marked UTF-16 string is a real_name finding in the payload scan.
