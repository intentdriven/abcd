---
schema_version: 1
id: "iss-2609251640464212"
slug: "the-no-verify-blockers-miss-the-same-bypass-spelled-as"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/gitconfig.go"
resolution: "A git command that sets core.hooksPath in its own text through -c, --config-env or the GIT_CONFIG environment is re-read with the no-verify flag after its subcommand, so the no-verify entries block a commit or push that moves its hooks; TestHooksPathOverrideIsNoVerify pins it."
impact: fix
resolved_by:
  commit: "5e05fad18d29bdafd3eeb8c547b9cc40eceb2e30"
---

The no-verify blockers miss the same bypass spelled as configuration: a commit or push that points core.hooksPath elsewhere for that one command, through -c, --config-env or the GIT_CONFIG environment, skips the repository hooks exactly as the no-verify flag does, and the matcher steps the -c value over unread. Found by review-guard finding 6 (pre-existing).

## Grounds

- pursued: every command-line spelling of a hooks-path override on commit or push blocks under the no-verify entries, while the same override on other subcommands and git config of the key stay allowed; an allow of an override spelling, or a block of status or log with it, would show it wrong
