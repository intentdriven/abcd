---
id: dcn-2611070734171590
slug: two-defects-in-the-terminal-drive-are
date: 2026-11-07
---
2026-11-07 — Two defects in the terminal drive are deferred past v0.17.0, each to be filed as an issue. One: the deferred read of a stop runs before signal handling closes (internal/surface/cli/drain_drive.go:596), so a person can be told the drive is stopping while a retry goes on to launch a live proof, and lanes after it if that start succeeds; another signal still ends it. Two: the refusal saying nothing started and nothing reserved can be false on a resuming start, because an earlier lane's reservation can still stand (drain_drive.go:255, :696). Grounds: both windows are narrow, neither loses work, and a second signal or the reservation's expiry clears them; the review at f4f1e560a asked for this record. Reverse by fixing them.
