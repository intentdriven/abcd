---
schema_version: 1
id: "iss-2611052230553388"
slug: "the-warning-for-an-agent-session-running"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "release cut preparation"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/off_profile.go"
remedy: "Write each warning to a record the person's status view reads, as well as to stderr, so a run's warnings survive until a person has seen them."
---

The warning for an agent session running without the wall is one line on stderr, which nothing keeps: in an unattended run the agent sees it and a person sees it only if the agent passes it on, so a person returning has no way to learn it fired.
