---
schema_version: 1
id: "iss-2610141706070227"
slug: "the-wait-flag-one-line-help-of-abcd"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement_load.go"
remedy: "Extend the flag summary to: and no other go test -race run, make preflight or earlier waiter is live, under any account."
---

The --wait flag one-line help of abcd implement load says it waits until the load is below the wait limit and no other go test -race run is live; the gate also stays shut while any account make preflight or an earlier waiter is live, as the command Long help and commands/implement.md say.
