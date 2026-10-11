---
schema_version: 1
id: "iss-2608270500200205"
slug: "abcd-guard-isshellfamily-omits-rbash-and"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/guard/match.go"
resolution: "guard descends into rbash and yash -c payloads (#353)"
impact: fix
---

abcd guard isShellFamily omits rbash and yash, so their -c payloads are never descended into and every blocker run through them is silently allowed. GitHub mirror: #353