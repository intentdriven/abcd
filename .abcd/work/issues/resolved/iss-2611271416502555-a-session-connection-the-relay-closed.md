---
schema_version: 1
id: "iss-2611271416502555"
slug: "a-session-connection-the-relay-closed"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/server_bounds_test.go"
remedy: "Keep the hijacked net.Conn under each peer and close it when the grace runs out, so abort ends the socket whatever the library is waiting on"
resolution: "the close grace now closes the hijacked connection under the WebSocket, so a close handshake waiting on a page that never answers ends at the grace"
impact: fix
---

A session connection the relay closed can outlive its close grace by the WebSocket library's five-second close-handshake wait, failing TestARejoinDuringASlowReplayStaysBounded on CI
