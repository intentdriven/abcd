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
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Should lane briefs carry sibling-branch file overlap, with generated surfaces regenerated on the merged tip (input to itd-2609201916151817)?"
---

Two lanes that each add a verb to the same surface conflict by construction: commands/intent.md, the CLI's dispatcher and its test file, docs/reference/cli/commands.md and release/surface.json all took every lane's edit, and every merge of main into a later lane resolved the same files again (lanes A and B on the intent page, B and C on the page and the lifecycle, C and D on the CLI file and the generated pages). Nothing warns a lane when a sibling branch edits the page it is about to edit, and the generated files double the conflict surface. Wanted, as an observation for the implement verb: a lane brief names the sibling branches touching its files (git can answer it at branch time), and the generated surfaces are regenerated on the merged tip rather than merged by hand.

## Evidence

- 2026-09-23, autonomous run A: the generated half made it worse. #671 regenerated the brief's surface chapters (itd-147's appendix), and #672 and #673, both queued with auto-merge armed, conflicted with the regenerated chapters at once; the forge disarmed their auto-merge, and update-branch could not bring them current. Each was re-landed by merging main and regenerating in the lane's worktree, then pushed as a new pull request (#679 and #678) with the old one closed. A third, #676, went DIRTY on later merges and took the same path to #681. Three of the run's thirty-three pull requests were replaced this way.
