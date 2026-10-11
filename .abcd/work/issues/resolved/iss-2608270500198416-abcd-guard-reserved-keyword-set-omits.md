---
schema_version: 1
id: "iss-2608270500198416"
slug: "abcd-guard-reserved-keyword-set-omits"
severity: "critical"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/guard/match.go"
resolution: "guard reaches command position through the bash coproc keyword (#318)"
impact: fix
---

abcd guard reserved keyword set omits bash coproc, so 'coproc git push --force' never reaches command position and every blocker run through coproc is silently allowed. GitHub mirror: #318