---
schema_version: 1
id: "iss-2610160608372082"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Add a discard bullet to the 11-history.md sub-verbs covering staged and quarantined files and --root-sha, and mention --root-sha on staged."
resolution: "11-history.md gains a discarding bullet (staged or quarantined file, confirmation, repository by root-commit SHA) and the staged listing now names its by-SHA form"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/11-history.md:255-256 mentions `abcd history discard` only as the confirmed discard of a quarantined file, and describes `staged` only as per-repo or all-repos. The shipped `history discard <staged-filename> --yes` deletes a staged or a quarantined raw transcript, and both `discard` and `staged` take --root-sha to reach a repository whose checkout is gone (commands/history.md:73-77, 190-236).
