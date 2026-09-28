---
schema_version: 1
id: "iss-2609261753290536"
slug: "the-foreign-owner-refusal-note-decides-whether-the-working"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/root.go"
resolution: "The refusal note's working-directory branch uses Lstat plus IsDir, matching readRepoLayer's refusal of a symlinked .abcd, so the note never says a layer is read that the loader refuses."
impact: fix
resolved_by:
  commit: "7b22cd16"
---

The foreign-owner refusal note decides whether the working directory's .abcd is read with os.Stat, which follows a symlinked cwd/.abcd, while readRepoLayer Lstat-refuses a symlinked .abcd; in that edge the note tells the user the working directory's configuration IS read when the loader refuses it. Use Lstat plus IsDir so the note and the loader agree.

## Grounds

- pursued: a refused root with a symlinked .abcd at the working directory gets a note saying NOT read, pinned by TestResolveRootRefusalNeverSaysASymlinkedAbcdIsRead; a note saying IS read for a .abcd that Load refuses would show it wrong
