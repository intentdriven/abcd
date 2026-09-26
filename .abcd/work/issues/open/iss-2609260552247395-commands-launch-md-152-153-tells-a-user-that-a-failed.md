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
---

commands/launch.md:152-153 tells a user that a failed release shows up as auto-release's detect job failing on the archive-pin gate, but auto-release runs no `launch archive` since 2026-09-25: the gate is release.yml's verify job, so the page sends the reader to the wrong job. Found by the v0.11.0 brief-surface cross-check (x-007).
