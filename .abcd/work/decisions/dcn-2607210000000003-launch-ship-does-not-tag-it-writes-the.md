---
id: dcn-2607210000000003
slug: launch-ship-does-not-tag-it-writes-the
date: 2026-07-21
---
2026-07-21 — `launch ship` does not tag. It writes the dated `## [X.Y.Z] - <date>`
heading in the reviewed ship PR; the unchanged `auto-release.yml` greps it on
merge and creates the tag. ADR-37 is preserved, not superseded: the reviewed ship
IS the release decision, and the bot-on-main alternative stays rejected.
