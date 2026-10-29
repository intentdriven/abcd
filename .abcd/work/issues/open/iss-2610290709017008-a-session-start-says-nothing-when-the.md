---
schema_version: 1
id: "iss-2610290709017008"
slug: "a-session-start-says-nothing-when-the"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "checking the SessionEnd cancellation issue's remedy against what merged"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/skew.go"
refines: ["iss-2610281259462058"]
remedy: "Rule whether a session start may ask for the newest release (today only abcd update --check does) and, if so, add a start-of-session notice naming the installed and newest versions and the update step, within the bootstrap's existing release lookup where it already runs."
---

A session start says nothing when the installed plugin is older than the newest published release, so a fix that has shipped (such as the SessionEnd staging change) never reaches a person whose plugin did not update, and nothing tells them to update. The start-of-session notice that exists compares the plugin surface with the binary serving it, and only reports a plugin newer than its binary; asking for the newest release is left to abcd update --check alone, which a session never runs.
