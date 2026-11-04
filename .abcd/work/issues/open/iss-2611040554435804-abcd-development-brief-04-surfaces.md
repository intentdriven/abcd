---
schema_version: 1
id: "iss-2611040554435804"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "State the persona exemption and its accepted residual in the rule's row."
---

.abcd/development/brief/04-surfaces/16-lint.md:160 describes privacy-hygiene as flagging absolute local paths on any tracked text line, with only the line waiver as an exemption. The rule also exempts a home path whose username segment is a registry persona other than this machine's own user (internal/core/repolint/rule_privacy.go:409, isPersonaHomeSegment), and accepts in a code comment that another person's home under such a name goes unflagged. The chapter states neither.
