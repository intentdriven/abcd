---
schema_version: 1
id: "iss-2610310331111109"
slug: "the-ahoy-chapter-s-user-scope-tree-which"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Bring the user-scope tree in 04-surfaces/01-ahoy.md to the full inventory 05-internals/03-configuration.md holds, and make the declaration-file count match the files it lists."
resolution: "the design brief now states what the binary does, in present tense"
impact: internal
---

The ahoy chapter's user-scope tree, which must agree with the configuration chapter, leaves out dogfood-roots, filter-roots, shell-path.json, tools.json and the identity files, and its closing sentence speaks of three declaration files while listing two.
