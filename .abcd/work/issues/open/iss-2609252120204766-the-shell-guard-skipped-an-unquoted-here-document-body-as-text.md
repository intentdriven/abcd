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
---

The shell guard skipped the body of a here-document whose delimiter is unquoted as text, but bash expands that body before the command reads it: a command substitution in it (dollar-paren, backticks, or one inside an arithmetic expansion) runs. Every blocker allowed when its command stood in such a substitution in the body, alone, behind a redirection or a list operator, or inside a substitution of its own (review4-guard finding 1). A quoted, escaped or partly escaped delimiter keeps the body literal.
