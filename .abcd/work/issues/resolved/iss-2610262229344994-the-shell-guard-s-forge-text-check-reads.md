---
schema_version: 1
id: "iss-2610262229344994"
slug: "the-shell-guard-s-forge-text-check-reads"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of fix/catch-ci-failures-before-push"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/forge.go"
remedy: "Walk a short-flag cluster as pflag does: skip gh's boolean shorthands, take a text flag's value from the rest of the word or the next word, and treat any other shorthand as consuming the rest; regression cases for -db, -dbx and -ab."
resolution: "forgeFlag walks a short-flag cluster as pflag does: gh's boolean shorthands are skipped, a text shorthand behind them is read with its value, and any other shorthand takes the rest of the word"
impact: fix
---

The shell guard's forge-text check reads a short text flag only from the second character of its word (forgeFlag, internal/core/guard/forge.go), so gh pr create -db 'text', pflag's cluster of --draft and --body, posts a body the check neither judges nor marks as unread.
