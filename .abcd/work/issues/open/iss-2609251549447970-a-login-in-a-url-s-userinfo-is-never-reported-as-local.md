---
schema_version: 1
id: "iss-2609251549447970"
slug: "a-login-in-a-url-s-userinfo-is-never-reported-as-local"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
---

A login in a URL's userinfo is never reported as local_username, for any login. localSuppressionSpans (internal/adapter/scanner/identity.go) suppresses the username matcher inside every URL span, so https://<login>@host/ and https://<login>:<password>@host/ pass the scanner with the caller's login intact; the userinfo is the one part of a URL that names an account rather than a resource, and a clone URL or a proxy setting quoted in a transcript carries it. Pre-existing: the suppression predates the scanner-cluster lane, whose review found it (item 6).
