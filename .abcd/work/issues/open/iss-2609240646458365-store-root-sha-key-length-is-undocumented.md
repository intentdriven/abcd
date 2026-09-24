---
schema_version: 1
id: "iss-2609240646458365"
slug: "store-root-sha-key-length-is-undocumented"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
---

The machine-scoped stores are documented as keyed on the repository's root commit (`~/.abcd/<store>/<root-sha>/`), and no page says the key is the full forty-character sha. The verbs that create a store (the run store, the history store, the transcript store) key on the full sha. The worktree store has no verb yet and is laid by hand, and its lanes sit under the eight-character short form; AGENTS.md says it is keyed "the way the history, transcript and voyage stores already are", which only a reader who checks those stores reads as the full sha. Autonomous run A's run file named the run store by the short form too, so the orchestrator appended its first events by hand to `~/.abcd/runs/<short-sha>/` while `abcd implement` wrote to the full-sha directory, and the two logs were joined by moving the short directory and leaving a symlink in its place. Wanted: AGENTS.md, the brief's configuration chapter and the worktree-store draft itd-2609091014076309 say the key is the full sha, and whatever lists the stores names a short-form directory beside a full one.
