---
schema_version: 1
id: "iss-2610230519116460"
slug: "abcd-session-setup-checks-the-relay-s"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "hosted-sites takeover, step 6 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/session.go"
related_intents: [itd-2610091016262335]
remedy: "judge the blueprint files with WriteBlueprint's read-only half before the app is created, or print the remaining steps before refusing on the blueprint, with a test that a blueprint refusal after creation still prints the install and settings links"
---

abcd session setup checks the relay's blueprint files only after GitHub has created the app, so a blueprint refusal (an edited Dockerfile or render.yaml) exits 2 before the steps that name the app's install and settings pages are printed, losing them on the one run that created the app (internal/surface/cli/session.go, found by the Sonnet review of spc-2610141709430146 step 6 on 2026-10-23)
