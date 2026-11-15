---
schema_version: 1
id: "iss-2611150658182921"
slug: "the-guard-s-proof-for-a-remote-branch"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/deleteproof.go"
remedy: "Answer unproven when git remote get-url --push origin and git remote get-url origin differ, or either gives more than one address."
---

The guard's proof for a remote branch delete reads origin's fetch address and does not hold when the push address differs.
