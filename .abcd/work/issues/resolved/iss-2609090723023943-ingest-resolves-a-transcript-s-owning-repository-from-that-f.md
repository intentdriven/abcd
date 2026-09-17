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
resolution: "Fixed on this branch. Ownership now resolves the session before the file: the main thread's recorded directory first, then any store already holding that session, and only then the file's own directory. A sub-agent whose worktree the harness removed on completion is therefore placed by its session rather than orphaned. Measured on a real session of 24 sub-agents, 19 of which had a vanished directory: all 24 resolved through the session, where per-file resolution would have discarded 19."
impact: fix
resolved_by:
  intent: "itd-2609091718566731"
  spec: "spc-2609091722230648"
---

Ingest resolves a transcript's owning repository from that file's own recorded working directory, which orphans sub-agents whose parent session is perfectly resolvable. A sub-agent that ran in an isolated worktree records that worktree as its working directory, and the harness removes the worktree when the agent stops, so the directory is already gone by the time anything reads the transcript. On this machine that is 51 worktree-isolated sub-agents whose parent transcript still exists and resolves cleanly, plus 191 more whose shared working directory has gone, plus 6 whose directory merely differs from the parent's. Every one of those files carries the parent session id, so ownership is derivable from the session even when the file's own directory is not. Resolving per file is the right rule for a main thread and the wrong authority for a sub-agent. The resolution should try the session first, through the parent transcript's working directory or any store already holding a record for that session id, and fall back to the file's own directory, so that orphan means the session cannot be placed rather than the file cannot. The same assumption reaches the capture hook, which resolves the payload's working directory the way the session-end hook does: when that directory is a removed worktree the detection fails and the hook exits zero having captured nothing, which loses exactly the implementation-lane agents whose work is most worth keeping.

## Grounds

- pursued: we expect the session to be the right authority for a sub-agent because every sub-agent transcript carries its parent session id, so the session places even when the file cannot; it is shown wrong if a sub-agent legitimately belongs to a different repository from its parent, which a worktree in another repo would produce
