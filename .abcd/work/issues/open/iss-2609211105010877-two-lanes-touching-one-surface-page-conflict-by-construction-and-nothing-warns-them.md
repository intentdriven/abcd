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
remedy: "Waits on the itd-2609201916151817 planning (ruling H): at claim time, list for each other local lane branch the files it changes since its merge base that the new lane's declared --path set also touches, and print them in the claim result and the lane brief; regenerate the generated surfaces (release/surface.json, docs/reference/cli/commands.md, the brief's surface chapters) on the merged tip in the land step instead of merging them by hand; prove it with a two-branch scratch-repository test that names the shared file."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Should lane briefs carry sibling-branch file overlap, with generated surfaces regenerated on the merged tip (input to itd-2609201916151817)?"
---

Two lanes that each add a verb to the same surface conflict by construction: commands/intent.md, the CLI's dispatcher and its test file, docs/reference/cli/commands.md and release/surface.json all took every lane's edit, and every merge of main into a later lane resolved the same files again (lanes A and B on the intent page, B and C on the page and the lifecycle, C and D on the CLI file and the generated pages). Nothing warns a lane when a sibling branch edits the page it is about to edit, and the generated files double the conflict surface. Wanted, as an observation for the implement verb: a lane brief names the sibling branches touching its files (git can answer it at branch time), and the generated surfaces are regenerated on the merged tip rather than merged by hand.

## Evidence

- 2026-09-23, autonomous run A: the generated half made it worse. #671 regenerated the brief's surface chapters (itd-147's appendix), and #672 and #673, both queued with auto-merge armed, conflicted with the regenerated chapters at once; the forge disarmed their auto-merge, and update-branch could not bring them current. Each was re-landed by merging main and regenerating in the lane's worktree, then pushed as a new pull request (#679 and #678) with the old one closed. A third, #676, went DIRTY on later merges and took the same path to #681. Three of the run's thirty-three pull requests were replaced this way.

## Remedy grounds (2026-09-29)

- `implement claim` already takes --path and refuses reading-corpus paths for a second session, and `abcd peers` already enumerates sibling branches, so overlap reporting extends two shipped seams.
- No outside-practice check: git answers the overlap directly and the regeneration targets exist.
- Rejected: a merge driver that auto-resolves generated files, which hides a stale generated file where regeneration is the truth.
