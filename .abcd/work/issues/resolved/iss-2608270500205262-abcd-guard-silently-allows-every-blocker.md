---
schema_version: 1
id: "iss-2608270500205262"
slug: "abcd-guard-silently-allows-every-blocker"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/guard (launcher tables)"
resolution: "guard inspects watch and GNU parallel single-string launcher payloads (#354)"
impact: fix
---

abcd guard silently allows every blocker run through watch '<cmd>' and GNU parallel: these single-string launchers pass their command to sh -c but are in no launcher table, so the quoted payload stays one opaque token. GitHub mirror: #354