---
schema_version: 1
id: "iss-2611070737112762"
slug: "a-drive-started-with-no-team-says"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Print those lines at the start."
---

A drive started with no team says nothing about it, and the run only stops days later when the first file it holds goes. The static wall check already returns the lines that would say so.
