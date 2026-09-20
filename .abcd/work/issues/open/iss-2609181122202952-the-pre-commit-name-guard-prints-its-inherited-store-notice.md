---
schema_version: 1
id: "iss-2609181122202952"
slug: "the-pre-commit-name-guard-prints-its-inherited-store-notice"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "Gropius managed-repo session gropiusllm-56, commits from a secondary worktree, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

The pre-commit name guard prints its inherited-store notice three times per commit from a linked worktree. The committed .githooks/pre-commit announces the itd-150 fallback once per run, at a single site ("pre-commit: abcd name-guard: inheriting the primary checkout's <store>" on stderr), so three copies on one commit mean the hook body ran three times for that commit, which points at the dispatch (the global hooks dispatcher plus the repo hook, or a pre-merge-commit run) rather than at the notice itself. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18 at v0.9.0, which called it cosmetic. Recorded because the repetition is also a measurement: if the guard runs three times, the scan costs three times what it should, and one run is the one whose refusal counts. Wanted: establish which invocations produce the three runs and either de-duplicate the dispatch or print the notice once per commit.
