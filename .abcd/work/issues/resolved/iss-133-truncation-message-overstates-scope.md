---
schema_version: 1
id: "iss-133"
slug: "truncation-message-overstates-scope"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "iss-112/114/116 review (2026-07-24 run queue, burst 9)"
found_at: "internal/core/lifeboat/sources_conventions.go"
resolution: "WalkFiles returns a typed walkTruncation (whole-walk cap stopped, depth-pruned count, oversized directories named) and the internals and open-questions adapters render one note per bound that fired; probe_truncation_test.go holds each bound's wording, with the whole-walk cap still saying the rest was not walked."
impact: fix
resolved_by:
  commit: "a95764d5"
---

the probe adapters' truncation message says the rest of the tree was not walked, but the per-directory read bound added for iss-112 can set truncated while the walk completed everything else; the over-warning is in the conservative direction, not a loud-staging violation, but the message is imprecise about WHAT was truncated

## Grounds

- pursued: a coverage note says which bound cut the scan and what it cut, and claims the rest of the tree went unwalked only when the walk actually stopped; shown wrong if a depth or per-directory truncation renders as the walk cap, which the tests assert
