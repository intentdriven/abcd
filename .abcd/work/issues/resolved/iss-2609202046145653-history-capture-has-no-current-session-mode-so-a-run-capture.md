---
schema_version: 1
id: "iss-2609202046145653"
slug: "history-capture-has-no-current-session-mode-so-a-run-capture"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "Dessau pilot run, session gropiusllm-64, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "history capture --session <id> --all [<path>...] stores the named session's main thread and every sub-agent transcript found under the paths or ingest_roots, placed as ingest places them. The current-session default is not built: no harness interface abcd reads exposes the running session id."
impact: additive
resolved_by:
  commit: "8a61f109"
---

history capture has no current-session mode, so a run captures its own transcripts by listing files by hand. At v0.9.0 the verb takes one transcript path, and bare abcd history capture with no file answers "--session <id> is required when reading from stdin"; nothing discovers the session that is running or the sub-agent transcripts it spawned. The Dessau pilot (session gropiusllm-64, 2026-09-20) captured its session and seven sub-agent transcripts with eight invocations after listing ~/.claude/projects/<escaped-cwd>/<session>/subagents/agent-*.jsonl by hand, which is exactly the path knowledge the store already has (list --session reaches a session and every sub-agent it spawned on the read side). Wanted: a write-side twin of that read, history capture --session <id> --all (the main thread and every sub-agent), with the current session as the default when the harness exposes its id, so a loop captures its own run in one call. Evidence for the implement verb (itd-2609201916151817), which would call it at the end of every lane.

## Grounds

- pursued: one call captures a session's main thread and all its sub-agents and nothing of another session; a sub-agent transcript of the named session left uncaptured, or another session's transcript stored, would show it wrong
