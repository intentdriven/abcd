---
schema_version: 1
id: "iss-2609231128235366"
slug: "the-capture-surface-chapter-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
resolution: "the capture surface chapter and the ledger README describe --lapsed-at as parked-optional for the lapse category, matching the verb and the command page"
impact: internal
resolved_by:
  commit: "9e66cc02"
---

The capture surface chapter (.abcd/development/brief/04-surfaces/06-capture.md, 'One flag is conditionally required') and the ledger README (.abcd/work/issues/README.md, '--lapsed-at (required with --category lapse)') state that a lapse capture without --lapsed-at exits 2 and writes nothing. It does not: the refusal is parked (iss-2609091009111294; the CLI comment in internal/surface/cli/cli.go says so, as does commands/capture.md and the same chapter further down), and 'abcd capture "..." --category lapse' with no --lapsed-at files the record at exit 0, confirmed in a scratch repository on 2026-09-23. Two prose surfaces describe a refusal the verb does not make.

## Grounds

- pursued: every prose surface agrees with the verb that a lapse capture without --lapsed-at is written; a surface still claiming exit 2 would show it wrong
