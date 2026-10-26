---
schema_version: 1
id: "iss-2610261152125261"
slug: "rs004-on-the-pull-request-form-every"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "CI-escape analysis, 2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/forge.go"
remedy: "In the shell guard's forge-text check, refuse a gh pr create or gh pr edit whose title or body names an issue id that no Refs: or Resolves: line in the title or body declares, judged with the Go twins of MENTION_RE and DECLARE_RE in internal/core/capture/mentions.go and pinned to scripts/check-issue-resolution.sh by a parity test; it runs only where .abcd/config.json carries outbound.declare_mentions: true, because RS004 is this repository's rule, not a bundled one"
---

RS004 on the pull-request form, every issue id the title and body name declared on a Refs: or Resolves: line, is checked only by the hosted attribution workflow, although the title and body exist before gh pr create runs and the shell guard already reads them for the outbound policy. Between 2026-10-18 and 2026-10-26 it failed 4 pull-request runs, each fixed by adding one line to the body.
