---
schema_version: 1
id: "iss-2610151422140743"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Rewrite .abcd/development/brief/06-delivery/02-verification-matrix.md:67 to match the shipped surface: capture.List (internal/core/capture/workflow.go:819) sorts by id only as a tiebreak and then calls prioritise(), which stably orders unblocked issues ahead of blocked ones and, within each group, higher severity first (workflow.go:1011-1016). The shipped…"
---

The brief at .abcd/development/brief/06-delivery/02-verification-matrix.md:67 says: Capture filtered list: `abcd capture list` with a state filter prints one line per issue, sorted by id. The shipped surface differs: capture.List (internal/core/capture/workflow.go:819) sorts by id only as a tiebreak and then calls prioritise(), which stably orders unblocked issues ahead of blocked ones and, within each group, higher severity first (workflow.go:1011-1016). The shipped order is derived priority, as 04-surfaces/06-capture.md §5 states, not id order. (v0.14.2 release crosscheck, false-claim, raised by opus in a38.)
