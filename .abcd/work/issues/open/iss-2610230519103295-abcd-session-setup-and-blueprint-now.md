---
schema_version: 1
id: "iss-2610230519103295"
slug: "abcd-session-setup-and-blueprint-now"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "hosted-sites takeover, step 6 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/session/blueprint.go"
related_intents: [itd-2610091016262335]
remedy: "when a stamped .abcd/relay/render.yaml exists, keep writing the Blueprint there and say so, or report it as superseded with the Blueprint Path to change on Render, with a test"
---

abcd session setup and blueprint now write the relay's Render Blueprint to render.yaml at the repository's root, but a team that deployed from the earlier location, .abcd/relay/render.yaml with Render's Blueprint Path set to it, keeps that stamped file untouched and unmentioned, so Render goes on syncing the old path while setup points at the Deploy button, which would create a second service (internal/core/session/blueprint.go, found by the Opus review of spc-2610141709430146 step 6 on 2026-10-23)
