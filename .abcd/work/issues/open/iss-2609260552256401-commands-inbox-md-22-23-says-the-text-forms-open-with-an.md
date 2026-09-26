---
schema_version: 1
id: "iss-2609260552256401"
slug: "commands-inbox-md-22-23-says-the-text-forms-open-with-an"
severity: "minor"
category: "inconsistency"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/inbox.md"
---

commands/inbox.md:22-23 says the text forms open with an `untrusted:` line, but bare `abcd inbox` opens with a header first and an empty inbox prints `abcd inbox - nothing waits` with no untrusted notice at all, so the framing a reader is told to rely on is second or absent. Found by the v0.11.0 brief-surface cross-check (x-064), reproduced by the classifier.
