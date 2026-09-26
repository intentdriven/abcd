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
---

The foreign-owner refusal note decides whether the working directory's .abcd is read with os.Stat, which follows a symlinked cwd/.abcd, while readRepoLayer Lstat-refuses a symlinked .abcd; in that edge the note tells the user the working directory's configuration IS read when the loader refuses it. Use Lstat plus IsDir so the note and the loader agree.
