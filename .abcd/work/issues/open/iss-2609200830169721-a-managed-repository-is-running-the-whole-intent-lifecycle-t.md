---
schema_version: 1
id: "iss-2609200830169721"
slug: "a-managed-repository-is-running-the-whole-intent-lifecycle-t"
severity: "minor"
category: "future-work-seed"
source: "agent-observation"
found_during: "Gropius sub-agent-lane experiment, session gropiusllm-2b, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md (planning interview)"
---

A managed repository is running the whole intent lifecycle through one sub-agent per record, and asks whether the pattern should become an abcd recipe. The shape, reported by the Gropius session gropiusllm-2b on 2026-09-20: a main session interviews the product thinker one decision at a time, reading the reviewer's recorded options off the draft rather than re-deriving them, records each answer as it is given, then fans out one Opus 5 sub-agent per intent or issue, each in its own worktree, which fills the draft from the answers, runs abcd intent plan, writes the spec, implements test-first, runs the security and ruthless reviewers as its own sub-agents, closes the spec in the landing pull request and arms auto-merge; record-only lanes edit exactly one record each and may not plan. The session named what abcd already gave it (quoted-text filing, capture provenance flags, intent ready --json as the gate every lane runs, the Review and Open Questions convention on a held draft) and what it lacked (recorded separately: the answer write path, the hold state, the append-lock on the decision log, the integration-branch landing, the lane worktree location). Whether abcd documents this as a recipe (a plugin page, a runbook in the brief, or a scaffolded lane brief) is the product thinker's to decide; recorded here so the decision has a record to rest on. Note that abcd's own record already plans the worktree half (itd-148) and asks the same session-per-lane question from the coordination side (itd-2609150819440345).
