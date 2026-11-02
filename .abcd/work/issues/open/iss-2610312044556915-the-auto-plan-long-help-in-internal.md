---
schema_version: 1
id: "iss-2610312044556915"
slug: "the-auto-plan-long-help-in-internal"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Add the covered and uncovered parts to the --auto-plan long help in one parenthesis, matching commands/build.md, and regenerate the CLI reference."
---

The --auto-plan long help in internal/surface/cli/build.go (about line 205) says only that a checkout that changed since the start hands the lane back, while commands/build.md says what the digest covers (HEAD, the index's entries and their flags, the files the ignore rules admit outside the local tier) and what it does not (ignored files, the git directory).
