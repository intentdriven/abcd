---
schema_version: 1
id: "iss-2609090723023943"
slug: "ingest-resolves-a-transcript-s-owning-repository-from-that-f"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "sub-agent transcript capture conceptual review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history"
---

Ingest resolves a transcript's owning repository from that file's own recorded working directory, which orphans sub-agents whose parent session is perfectly resolvable. A sub-agent that ran in an isolated worktree records that worktree as its working directory, and the harness removes the worktree when the agent stops, so the directory is already gone by the time anything reads the transcript. On this machine that is 51 worktree-isolated sub-agents whose parent transcript still exists and resolves cleanly, plus 191 more whose shared working directory has gone, plus 6 whose directory merely differs from the parent's. Every one of those files carries the parent session id, so ownership is derivable from the session even when the file's own directory is not. Resolving per file is the right rule for a main thread and the wrong authority for a sub-agent. The resolution should try the session first, through the parent transcript's working directory or any store already holding a record for that session id, and fall back to the file's own directory, so that orphan means the session cannot be placed rather than the file cannot. The same assumption reaches the capture hook, which resolves the payload's working directory the way the session-end hook does: when that directory is a removed worktree the detection fails and the hook exits zero having captured nothing, which loses exactly the implementation-lane agents whose work is most worth keeping.
