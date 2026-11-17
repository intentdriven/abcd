---
schema_version: 1
id: "iss-2611170410462663"
slug: "while-the-person-is-away-the-stop-check"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "[redacted-user] session waiting on the v0.17.0 tag"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Revisit dcn-2610181035355741 for an unpushed branch the checking session did not make and no live session holds: name it once per session as waiting on the person, and keep blocking the stop on the session's own owed work, so no session can exempt work of its own"
---

While the person is away, the stop check keeps blocking every turn end on a push that is neither the checking session's to make nor any live session's, so a session with nothing else left answers the same line turn after turn. From 03:30 GMT on 2026-11-17 the [redacted-user] session's stop check named branch records-run-1104b, which another session made and whose push is the person's act, at every turn end for over an hour; queuing it with abcd mode queue at 03:44 changed nothing, and once "the set has not moved over two wakes" a wake no longer released the stop either, so the session spent dozens of turns repeating itself while its own work waited on a release tag.

Part of this is decided. dcn-2610181035355741 keeps a queued push owed by default and lets a queued step hand an item back only under --hold-pushes (handBack, internal/core/autonomy/owedread.go), so that a session cannot exempt itself; it expects a session with no instruction to push to end at the host's cap with the stop recorded for the person. What the decision does not reach is a branch the checking session did not make: even under --hold-pushes, handBack hands back only an own branch, so a peer's unpushed branch stays owed to every session that checks, and each pays for it in repeated turns rather than once.
