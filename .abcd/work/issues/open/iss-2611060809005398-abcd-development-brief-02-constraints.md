---
schema_version: 1
id: "iss-2611060809005398"
slug: "abcd-development-brief-02-constraints"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Correct the passage to say what the code does."
refines: [iss-2611040554013625]
---

`.abcd/development/brief/02-constraints/04-naming.md:14`: The naming row's list of ahoy write paths omits the worktree and shared-folder sub-verbs that write. Three more ahoy sub-verbs write, by their own help sentences: `abcd ahoy worktree add` writes the worktree, its start record and its local tier; `abcd ahoy worktree prune` writes the notes archive and removes the worktree and its branch; `abcd ahoy shared-folder <folder>` writes the choice.
