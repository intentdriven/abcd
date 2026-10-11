---
schema_version: 1
id: "iss-2609091647582259"
slug: "the-memory-board-hides-its-staleness"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/bare.go"
resolution: "The bare memory board prints every drift line on the text render in the words the --json drift list carries, and each line names abcd memory ingest as the verb that rebuilds the stale file; a current store prints none."
impact: fix
resolved_by:
  commit: "989f69a8d"
---

The memory store's bare status carries a drift field that says the index is stale and an ingest should be run. It is set on the result, it is emitted under the JSON envelope, and the human render never prints it, so the only reader who can act on the warning is the one who asked for machine output. The person who typed the bare verb to see how the store is doing is shown everything except the one line that asks them to do something. That is the loud-staging principle inverted: a degraded state that announces itself to a parser and stays quiet to a person, which is the shape the principle exists to refuse, and it is worse than silence because the board looks complete. Fix direction: print the drift line in the text render beside the counts it already shows, in the same words the JSON carries, so the two surfaces say one thing. Detector: a store whose index is stale renders the staleness on the bare human board as well as in the JSON envelope, and a store that is current renders neither.

## Grounds

- pursued: a store with no index.md shows the index-stale line on the text board and a freshly ingested store shows no stale line on either surface; a stale store whose text board omits a line the JSON carries would show it wrong
