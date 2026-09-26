---
schema_version: 1
id: "iss-2609252120204766"
slug: "the-shell-guard-skipped-an-unquoted-here-document-body-as-text"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "An unquoted here-document body is read as bash expands it: each command substitution in it (dollar-paren, backtick, inside an arithmetic expansion) is followed as commands of the redirecting command's chain, its text stays data, and a backslash in the delimiter quotes it."
impact: fix
resolved_by:
  commit: "b0fc5c66"
---

The shell guard skipped the body of a here-document whose delimiter is unquoted as text, but bash expands that body before the command reads it: a command substitution in it (dollar-paren, backticks, or one inside an arithmetic expansion) runs. Every blocker allowed when its command stood in such a substitution in the body, alone, behind a redirection or a list operator, or inside a substitution of its own (review4-guard finding 1). A quoted, escaped or partly escaped delimiter keeps the body literal.

## Grounds

- pursued: every blocker run from an unquoted body blocks and a quoted, escaped or partly escaped body allows; a blocked command in a body substitution that allows, or an everyday heredoc (a commit message with apostrophes, a date in a document) that blocks, would show it wrong
