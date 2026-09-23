---
schema_version: 1
id: "iss-2609200832054807"
slug: "no-verb-moves-an-untracked-new-record-into-a-lane-worktree-a"
severity: "nitpick"
category: "future-work-seed"
source: "agent-observation"
found_during: "Gropius sub-agent-lane experiment, session gropiusllm-2b, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/create.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: move or file an uncommitted record into a named lane worktree; the product thinker routes). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

No verb moves an untracked new record into a lane worktree. A draft filed with abcd intent "<text>" in the primary checkout is an untracked file with no history, so the one legitimate copy case in a worktree-per-lane flow, handing that draft to the lane that will plan and implement it, has no write path: the lane copies the file by hand, and the copy and the original are two untracked files with the same id in two trees until one is deleted, which is the shape the sibling-worktree visibility intent (itd-2609091416295622) exists to notice. Relayed from the Gropius session gropiusllm-2b on 2026-09-20 (sub-agent-lane experiment, low priority in the session's words). Wanted, either: a record export/import pair (or a move) that carries one uncommitted record from this checkout into a named worktree and removes it here, or filing directly into a named worktree (abcd intent "<text>" --worktree <lane>), so the record has one home from its first byte. Intent-shaped; the routing is the product thinker's.
