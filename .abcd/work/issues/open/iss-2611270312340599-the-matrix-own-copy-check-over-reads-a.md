---
schema_version: 1
id: "iss-2611270312340599"
slug: "the-matrix-own-copy-check-over-reads-a"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the matrix glob lane, rounds 5 and 6, 27/11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/matrix/compliance.go"
remedy: "Count a code span's climbing paths against a bound of their own, apart from the pattern bound. End a chmod's reach at its own code span unless the span holds a target. Settle quoting with a shell-word reader for code spans rather than the both-ways reading, and keep a test per case."
---

The matrix own-copy check over-reads a code span on a prose line, so a copy can be marked unexamined when it stayed in its clone. First, every distinct climbing path in a code span takes one glob walk from the 100-pattern bound in internal/core/matrix/compliance.go, so a copy past 100 such paths reads unexamined. Second, a code span holding only the word chmod keeps the mode change open to the line's end, and a chmod in a code span on a line with an apostrophe or a backslash carries past its span. Either way a later .. on that line is judged as the chmod's target. Every case fails closed: the copy reads unexamined, never kept.
