---
schema_version: 1
id: "iss-2609260552247395"
slug: "commands-launch-md-152-153-tells-a-user-that-a-failed"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
resolution: "commands/launch.md and the brief's launch chapter name release.yml's verify job as where the archive-pin proof refuses, before the tag on the auto-release path."
impact: fix
resolved_by:
  commit: "3583b3aa"
---

commands/launch.md:152-153 tells a user that a failed release shows up as auto-release's detect job failing on the archive-pin gate, but auto-release runs no `launch archive` since 2026-09-25: the gate is release.yml's verify job, so the page sends the reader to the wrong job. Found by the v0.11.0 brief-surface cross-check (x-007).

## Grounds

- pursued: the page's archive-pin failure bullet names the job that runs launch archive --verify; a step named Plugin archive reproduces the committed pin in auto-release.yml, or none in release.yml's verify job, would show it wrong
