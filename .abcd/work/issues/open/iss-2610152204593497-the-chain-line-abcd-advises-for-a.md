---
schema_version: 1
id: "iss-2610152204593497"
slug: "the-chain-line-abcd-advises-for-a"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim.go"
remedy: "Only run the copy when rev-parse succeeded: d=$(git rev-parse --git-common-dir) && h=\"$d/hooks/abcd-commit-msg\" && [ -x \"$h\" ] && { \"$h\" \"$1\" || exit 1; }"
---

The chain line abcd advises for a commit-msg hook it did not write falls back to /hooks/abcd-commit-msg when git rev-parse --git-common-dir fails, so a file at that root path would run; no gain to an attacker was found, as only root or the same user can create it.
