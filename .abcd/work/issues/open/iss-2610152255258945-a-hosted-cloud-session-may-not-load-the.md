---
schema_version: 1
id: "iss-2610152255258945"
slug: "a-hosted-cloud-session-may-not-load-the"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "build of spc-2610120941441282 step 4, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".claude/settings.json"
remedy: "Prove it in the spec's live step 12; if the plugin does not load, have the connector's setup step install it from the checkout's own marketplace, or put the choice back to the product thinker"
---

A hosted cloud session may not load the abcd plugin that the repository's committed settings file enables: the host's cloud-environments page says cloud sessions do not install plugins declared in a repository's settings, so the plugin half of the cloud-sessions intent's criterion C11 (decision 10) may have no effect there, while the committed hooks and go run still work
