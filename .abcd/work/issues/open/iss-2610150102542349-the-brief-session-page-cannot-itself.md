---
schema_version: 1
id: "iss-2610150102542349"
slug: "the-brief-session-page-cannot-itself"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "hosted-sites final security review (finding D)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "Check the client id's app kind when GitHub documents a stable signal (a client-id format or a public lookup by client id); until then the relay trust anchor (relays.json, or the person's confirmation) is the control, as the page's comment says"
---

The brief session page cannot itself tell a GitHub App's client id from an OAuth app's, so a relay the repository names or the person confirmed could report app_type github-app while holding an OAuth app; with no scope requested a fresh grant reaches public data only, but a person who earlier granted that OAuth app wider scopes is signed in again silently with them
