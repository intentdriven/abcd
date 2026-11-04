---
schema_version: 1
id: "iss-2611040554457784"
slug: "the-session-chapter-says-the-blueprint"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/session/blueprint.go"
remedy: "Set firstSessionRelease to the first release that carried the verb, and add both refusals to the chapter's blueprint section."
---

The session chapter says the blueprint refuses outside a checkout abcd manages and for a release that publishes no checksum for the linux/amd64 binary (.abcd/development/brief/04-surfaces/37-session.md:127-129). It also refuses a release older than a floor, naming firstSessionRelease (internal/core/session/blueprint.go:99, refusal at :466), and a checkout whose origin names no repository on github.com. The chapter states neither. The floor is itself stale: its comment says the cut that first ships the verb sets it, yet it reads v0.17.0 while the changelog's 0.16.0 section (CHANGELOG.md:85) already describes the session page, so the refusal can direct a person to a release that does not exist.
