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
resolution: "utf16RuneAt decodes a valid surrogate pair as one character outside the Basic Multilingual Plane and lets the byte-order-marked run continue past it, so a name after an emoji in one UTF-16 string is read; a lone surrogate, a pair spelling a noncharacter and every unit utf16TextRune refuses still end the run. Test: TestUTF16RunContinuesPastAnAstralCharacter (both byte orders, plus the lone-surrogate case)."
impact: fix
resolved_by:
  commit: "1b4d15c0e"
---

The UTF-16 byte view (internal/adapter/scanner/utf16.go, utf16TextRune) ends a run at a surrogate, so a character outside the Basic Multilingual Plane, an emoji written as a surrogate pair, cuts a byte-order-marked UTF-16 string in two, and a name after it is never read when fewer than two code units stood before the pair: an emoji, a space and a name behind a byte-order mark yields no view at all, while the same name before the emoji is found. Detector: the caller's name after an emoji in one byte-order-marked UTF-16 string is a real_name finding in the payload scan.

## Grounds

- pursued: a name after an astral character in one marked UTF-16 string is found without new findings on chance bytes; a launch dry-run that reports a new finding on this tree's assets, or a marked name after an emoji the scan still passes, would show it wrong
