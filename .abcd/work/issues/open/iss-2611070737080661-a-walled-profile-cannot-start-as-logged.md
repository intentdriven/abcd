---
schema_version: 1
id: "iss-2611070737080661"
slug: "a-walled-profile-cannot-start-as-logged"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Have the login step or the start set the profile up so the check passes (for example place the sync outside the walled profile), or find the harness's own switch for turning the skills sync off; prove on a live start whether the harness syncs on every launch."
---

A walled profile cannot start as logged in. The harness syncs the account's skills into a synced folder under the profile's skills directory at login (hundreds of files), and the wall's check refuses a skills directory with anything in it ('the profile directory holds skills, which a launch would load; move it out'). Every new user of a drive meets this at their first start.
