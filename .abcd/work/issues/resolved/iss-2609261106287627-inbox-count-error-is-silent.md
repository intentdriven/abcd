---
schema_version: 1
id: "iss-2609261106287627"
slug: "inbox-count-error-is-silent"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cutfix item 2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/report.go"
resolution: "inboxGreeting and boardInbox name a Count failure in one line on stderr — among the session-start notices, and beside the board's render in text and JSON — in place of nothing; stdout keeps counts only."
impact: fix
resolved_by:
  commit: "00c1c65ac4f4c2a46a7e9aa85fce80f64fc2c92e"
---

A refused or unreadable inbox is silent at session start and on the board: inboxGreeting and boardInbox (internal/surface/cli/report.go) call report.Count and discard its error, so an inbox abcd refuses to read prints nothing where a count belongs, which reads as an empty inbox. Loud staging asks for one line naming the refusal, sanitised and home-redacted, in place of nothing.

## Grounds

- pursued: with the inbox a symlink, session start and the board each print one home-redacted line naming the refusal on stderr and no count on stdout; silence, or the refusal text on the session-start stdout, would show it wrong
