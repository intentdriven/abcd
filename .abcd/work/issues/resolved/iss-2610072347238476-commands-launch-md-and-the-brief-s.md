---
schema_version: 1
id: "iss-2610072347238476"
slug: "commands-launch-md-and-the-brief-s"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "review of the release-check lane (run-2610072114599080), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Say in commands/launch.md and the brief's launch chapter that the unfixed-finding refusal also names an open issue record that differs from HEAD (an uncommitted edit, waiver or regrade), and render the uncommitted paths in the findings line when unfixed or deleted findings are also present, so one run names every reason."
resolution: "The findings line now names the uncommitted open records when an unfixed or deleted finding is present too, and commands/launch.md and the brief's launch chapter say the unfixed-finding refusal also covers an open record that differs from HEAD."
impact: fix
resolved_by:
  commit: "6559f9afd5b293d117291491f2266f621c7dd7d2"
---

commands/launch.md and the brief's launch chapter describe the unfixed-finding refusal as major/critical-only, but the findings gate now also emits it for any open issue record that differs from HEAD; and when unfixed or deleted findings are present too, the uncommitted paths appear only in --json, so the operator loops once more before seeing them.

## Grounds

- pursued: a cut with an unfixed or deleted finding and an uncommitted open record renders every uncommitted path on its findings line, so one run names every reason; a findings line in that case without the paths would show it wrong
