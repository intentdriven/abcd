---
schema_version: 1
id: "iss-2610262033126804"
slug: "itd-39-names-a-workspace-memory-scope"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 374 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:547"
remedy: "In itd-39 replace the 'workspace' scope with the repo scope (repo narrower than user) in the press release, scope bullets and acceptance criteria; 07-memory.md section 0 already uses repo and user."
related_intents: [itd-39]
finding_ref: cfk-f010b01c44681ada
---

itd-39 names a 'workspace' memory scope; the ahoy chapter says abcd has no workspace layer, only the repository and the user scope

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:547` — "There is **no workspace, host, or development-environment layer.**"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35` — "On a recall-keyword conflict, the narrower scope wins (workspace > user)."

itd-39 queries memory across two scopes, the workspace and the user's personal store, and `<workspace>/.abcd/memory/` wins a conflict over the user scope. The ahoy chapter says abcd manages exactly one kind of folder, a repository, plus one user-scope directory. It says there is no workspace, host or development-environment layer, and that a folder grouping repositories gets no special treatment. Calling the narrower scope 'workspace' either names the repository with a word the brief rules out or implies a layer the brief says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 374.
