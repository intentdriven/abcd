---
schema_version: 1
id: "iss-2610050859363813"
slug: "dashboard-start-killed-the-group-of-a-server-that-had-exited"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "PR #828 CI, Linux race lane, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/control.go"
remedy: "Give a failed configuration write the exit grace a closed readiness pipe gets, and have start reap the server itself with wait4(WNOHANG) so it signals a group only while the server is unreaped."
---

On the Linux CI race lane, TestStartNeverSignalsAServerThatAlreadyExited failed in 0.03s: dashboard start killed the process group of a server that had already exited. A server that exits at once closes its configuration pipe before start writes to it; on Linux the write fails with EPIPE, and start treated that error as a stuck server and killed its group at once, while a concurrent cmd.Wait could already have reaped it, so the group number could by then name another program's processes. macOS passed only because the write lands in the pipe first.
