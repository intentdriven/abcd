---
schema_version: 1
id: "iss-2608270500192596"
slug: "abcd-guard-tokenizer-has-no-backtick"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/guard/tokenize.go"
resolution: "guard blocks a hazard wrapped in backtick command substitution (#312)"
impact: fix
---

abcd guard tokenizer has no backtick command-substitution case, so a blocker inside backticks (e.g. `gh repo delete ...`) passes unflagged. GitHub mirror: #312