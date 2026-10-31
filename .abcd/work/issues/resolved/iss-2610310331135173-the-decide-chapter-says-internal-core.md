---
schema_version: 1
id: "iss-2610310331135173"
slug: "the-decide-chapter-says-internal-core"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/24-decide.md"
remedy: "Reword 04-surfaces/24-decide.md to say the site derives handles with its own pattern while importing recordid for its paths."
resolution: "the design brief now states what the binary does, in present tense"
impact: internal
---

The decide chapter says internal/core/site does not import core/recordid at all; it does (project.go), though it still derives handles with its own pattern.
