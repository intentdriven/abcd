---
schema_version: 1
id: "iss-2611060809113118"
slug: "04-surfaces-36-dashboard-md-3-calls-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/36-dashboard.md"
remedy: "Name the set-up flow's short-lived loopback listener in the session chapter and word the count as two served listeners plus that one."
---

`04-surfaces/36-dashboard.md:3` calls the dashboard one of abcd's two network listeners, the other being the brief session relay, and `04-surfaces/37-session.md:14`, `06-delivery/02-verification-matrix.md:25` and `05-internals/03-configuration.md:1914` count the same two. Session set-up opens a third: `LoopbackListener` (`internal/surface/session/appsetup.go:90`, called at `internal/surface/cli/session.go:821`) listens on a loopback port for the forge's answer to the app manifest. The session chapter does not describe it, and the listener boundary test admits two packages rather than two listeners.
