---
schema_version: 1
id: "iss-2610230519122686"
slug: "when-abcd-session-setup-gives-up-after"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "hosted-sites takeover, step 6 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/appsetup.go"
related_intents: [itd-2610091016262335]
remedy: "on a final failure name the organisation's or person's GitHub Apps settings page to check for the app, and accept an existing app's client id (for example abcd session setup --client-id) so it can be recorded without registering another"
---

When abcd session setup gives up after three failed exchanges with GitHub for the new app, GitHub may already have created it, and the refusal names neither the app nor a way to record its client id, so a rerun registers a second app and the first must be found and deleted by hand (internal/surface/session/appsetup.go, from the Opus and Sonnet reviews of spc-2610141709430146 step 6 on 2026-10-23; the retry itself is fixed)
