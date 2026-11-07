---
schema_version: 1
id: "iss-2611061000134366"
slug: "a-person-with-two-subscriptions-cannot"
severity: "major"
category: "ux"
source: "agent-observation"
found_during: "two subscriptions as profiles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/profile.go"
remedy: "Let abcd drain --drive --profile <name> choose among the directories by name, applying every check the one-directory path applies, and hold a run to the profile it started on."
refines: [itd-2611040344170206]
resolution: "abcd drain --drive --profile <name> chooses among the directories by name, under every check the one directory takes, and a run keeps the profile it started on."
impact: additive
---

A person with two subscriptions cannot keep both as profiles: a second directory under the profiles folder makes every abcd drain --drive refuse, naming a draft intent, so the only way to run on either is to move the other out.
