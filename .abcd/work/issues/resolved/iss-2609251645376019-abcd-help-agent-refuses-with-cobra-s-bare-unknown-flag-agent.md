---
schema_version: 1
id: "iss-2609251645376019"
slug: "abcd-help-agent-refuses-with-cobra-s-bare-unknown-flag-agent"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "abcd help --agent now refuses with the same message as abcd --agent, naming abcd --help --agent, exit 2: applyHelpVerbAgentRefusal wraps the root's flag-error function, which cobra's lazily built help verb inherits, and one constant carries the refusal for both paths."
impact: fix
resolved_by:
  commit: "8cf52a722"
---

abcd help --agent refuses with cobra's bare 'unknown flag: --agent' (exit 2), while abcd --agent refuses naming the spelling that works; the help-subcommand path does not name --help --agent (review-helpgroups 3).

## Grounds

- pursued: we expect every path that takes --agent without --help to name abcd --help --agent; a help-verb invocation still showing cobra's bare unknown-flag line would show it wrong
