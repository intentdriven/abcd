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
resolution: "commands/inbox.md and the brief's inbox chapter place the untrusted line before the first word a report wrote, as the list and show render it, and say an empty list prints none."
impact: fix
resolved_by:
  commit: "180c812e"
---

commands/inbox.md:22-23 says the text forms open with an `untrusted:` line, but bare `abcd inbox` opens with a header first and an empty inbox prints `abcd inbox - nothing waits` with no untrusted notice at all, so the framing a reader is told to rely on is second or absent. Found by the v0.11.0 brief-surface cross-check (x-064), reproduced by the classifier.

## Grounds

- pursued: the page's framing claim matches the text forms; a list or show text output printing a report's words before the untrusted line (TestInboxListAndShowFrameReportsAsData) would show it wrong
