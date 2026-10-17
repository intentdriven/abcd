---
schema_version: 1
id: "iss-2610161015410527"
slug: "drain-lanes-never-merge-the-default"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Regenerate the generated files after merging (for example a regenerating merge driver) so a lane's merge with main never stops on them (grounds: lockfile merge drivers)"
refines: [iss-2610150526192138]
related_intents: [itd-2610161016391855]
resolution: "an issue lane's sync fetches and merges origin's default branch whenever it moved, before the records and the push, a fresh round judging the merge; a conflict on declared generated files goes to an implementer to regenerate, any other hands the lane back naming the files with nothing pushed, and a pull request that starts to conflict on generated files alone enters the sync again"
impact: fix
resolved_by:
  commit: "41b9add2ab39f533d10db45becea954c7a8a2f4d"
---

Drain lanes never merge the default branch before their pull request, and the footprint taken from found_at never names generated files, so lanes from several accounts conflict on generated files: of 259 merges to main since 2026-10-01, 54 touched docs/reference/cli/commands.md, 29 surface.json and 18 the brief's surfaces README.
