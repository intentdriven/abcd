---
schema_version: 1
id: "iss-2609211105010877"
slug: "two-lanes-touching-one-surface-page-conflict-by-construction-and-nothing-warns-them"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md; internal/surface/cli/cli.go"
---

Two lanes that each add a verb to the same surface conflict by construction: commands/intent.md, the CLI's dispatcher and its test file, docs/reference/cli/commands.md and release/surface.json all took every lane's edit, and every merge of main into a later lane resolved the same files again (lanes A and B on the intent page, B and C on the page and the lifecycle, C and D on the CLI file and the generated pages). Nothing warns a lane when a sibling branch edits the page it is about to edit, and the generated files double the conflict surface. Wanted, as an observation for the implement verb: a lane brief names the sibling branches touching its files (git can answer it at branch time), and the generated surfaces are regenerated on the merged tip rather than merged by hand.
