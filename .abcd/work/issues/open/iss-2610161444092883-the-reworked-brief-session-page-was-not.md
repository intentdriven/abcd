---
schema_version: 1
id: "iss-2610161444092883"
slug: "the-reworked-brief-session-page-was-not"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "With a GitHub App that has its device flow on and is installed on a test repository, run abcd session start on a tailnet machine and a cloud relay with relays.json, and record a live capture of: signing in by code and by redirect, starting a session that opens on the press release, a guest joining from a second device, a suggestion applied, and the pull request sent; fix whatever breaks."
---

The reworked brief session page was not proven end to end against real GitHub: no device-flow or web-flow sign-in, hosting, joining, editing or sending back ran in a browser, because the build sandbox reached neither github.com's sign-in endpoints nor a real GitHub App. Headless screenshots of the start screens on loopback and Go tests against a fake GitHub are the only evidence.
