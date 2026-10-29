---
id: dcn-2610222222452960
slug: decided-by-the-technical-facilitator-on
date: 2026-10-22
---
2026-10-22 — Decided by the technical facilitator on delegation (coordinator ruling, 2026-10-22): the sweep command's liveness probe (itd-2610222214476887) reads /proc/<pid>/cwd on Linux and runs /usr/sbin/lsof by absolute path on macOS, with a deadline, a bounded read and defensive parsing, never cgo, because the released binaries are cross-compiled per platform; it fails closed, so when lsof is missing, times out or cannot be read every worktree it could not clear counts as possibly live and is left in place, named with the reason.
