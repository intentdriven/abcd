---
schema_version: 1
id: "iss-2608270500198285"
slug: "abcd-guard-matchsegment-commandof"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/guard/match.go"
resolution: "guard matches the command basename case-insensitively, closing the macOS case-variant bypass (#315)"
impact: fix
---

abcd guard matchSegment/commandOf compares command basename byte-exact, so GIT push --force and other case-variant spellings bypass the matcher on a case-insensitive macOS filesystem. GitHub mirror: #315