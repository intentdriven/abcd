---
schema_version: 1
id: "iss-2611061913041202"
slug: "abcd-drain-drive-refuses-at-its-live"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "drive start on harness 2.1.319"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "read the approver from the status the harness reports for its one server, admit only the harness's own built-in plugins, make the proof's working directory a repository whose commit is the approver's base, and say the harness's version in a wall refusal"
---

abcd drain --drive refuses at its live proof on a harness whose start-up report no longer lists server tools and lists plugins of its own, and the proof's approver judges against a base that is no commit
