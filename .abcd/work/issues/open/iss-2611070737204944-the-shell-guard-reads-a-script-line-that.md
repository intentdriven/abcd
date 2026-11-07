---
schema_version: 1
id: "iss-2611070737204944"
slug: "the-shell-guard-reads-a-script-line-that"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Teach the script reading that -trimpath after a build verb is a go flag, and add a test with the variable form."
refines: [iss-2609281134544802]
---

The shell guard reads a script line that runs go through a variable with the build -trimpath flag as killall -t (script reading, kill by pattern), so any script that runs go that way is blocked.
