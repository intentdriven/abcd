---
schema_version: 1
id: "iss-2610121653432051"
slug: "abcd-implement-report-json-echoes-the"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/report.go"
remedy: "Mask every log-derived string Compare puts on the Report (mode, session, role, leader and the rest) with termsafe.Sanitize where Compare composes it, as listField now does for the outage spans, with a test parsing a log whose lines carry U+202E and U+009B in each field and asserting nothing raw on implement report --json."
resolution: "Every log-derived string implement report carries (each mode and so the leader, each session's id and role, each intervention's kind, and the log an unparsed line names) is masked with termsafe.Sanitize where Compare composes the report, as listField already masks the outage spans."
impact: fix
resolved_by:
  commit: "39832ed54056bf0a0e00e7307bc78c6506b25e9a"
---

abcd implement report --json echoes the run log's text raw beyond the outage spans: each mode tally's mode, each session context's session and role, and the other event strings Compare reads from the log the sessions append to, which ParseLog checks only for presence. A hand-edited or repository-shipped run log, or one named with --log, whose line carries a control, bidi or zero-width rune puts it on the stream. Found while masking the outage spans for iss-2610121542263820, which closed only the span fields.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
