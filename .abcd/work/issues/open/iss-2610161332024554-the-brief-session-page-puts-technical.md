---
schema_version: 1
id: "iss-2610161332024554"
slug: "the-brief-session-page-puts-technical"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "live test of the hosted-sites draft, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "Never ask a person for a relay address: the host's start command prints a ready link that carries it, and the repository's relay list fills it otherwise; make GitHub sign-in the default with the token tucked away as a fallback; open on the brief's first file; rewrite the screen's words for a product thinker."
---

The brief session page puts technical setup in front of a product thinker before they can see the brief: its first screen asks for a relay address (a term the page never explains), then for a GitHub personal access token, and the brief appears only after hosting and picking a file. Tested by the product thinker on the hosted-sites draft, who could not tell what the page was for or what to enter. A person who only wants to work on the brief should arrive with the relay already chosen and sign in with one GitHub button.
