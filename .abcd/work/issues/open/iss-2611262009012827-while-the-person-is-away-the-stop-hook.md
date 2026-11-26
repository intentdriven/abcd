---
schema_version: 1
id: "iss-2611262009012827"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Let the owed-work reader take a branch as landed when its tree changes are contained in the default branch (a merge-tree or diff check against the merge base, not only patch ids), and give the session a verb to mark a lane branch spent with a reason so the hook stops counting it without deleting the branch."
---

While the person is away, the stop hook holds a session's turn open on lane branches whose work already landed through an integration branch that rewrote their commits (squashed, reworded or re-split), because their patches do not match origin's default branch and the hook cannot tell them landed; it repeats the same six items every wake and says a wake no longer releases the stop, though nothing is owed.
