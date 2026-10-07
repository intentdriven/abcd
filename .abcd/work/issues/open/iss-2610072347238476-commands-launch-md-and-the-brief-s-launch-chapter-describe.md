---
schema_version: 1
id: "iss-2610072347238476"
slug: "commands-launch-md-and-the-brief-s-launch-chapter-describe"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "review of the release-check lane (run-2610072114599080), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Say in commands/launch.md and the brief's launch chapter that the unfixed-finding refusal also names an open issue record that differs from HEAD (an uncommitted edit, waiver or regrade), and render the uncommitted paths in the findings line when unfixed or deleted findings are also present, so one run names every reason."
---

commands/launch.md and the brief's launch chapter describe the unfixed-finding refusal as major/critical-only, but the findings gate now also emits it for any open issue record that differs from HEAD; and when unfixed or deleted findings are present too, the uncommitted paths appear only in --json, so the operator loops once more before seeing them.
