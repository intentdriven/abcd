---
id: dcn-2609300000000023
slug: a-command-line-runner-admits-a-harness
date: 2026-09-30
---
2026-09-30 — A command-line runner admits a harness reached through a directory, or a binary, that is group-writable only when the group is the system administrator group (gid 0 anywhere, gid 80 `admin` on darwin) and other cannot write it; other-writable stays refused whatever the group, and every other group stays refused (lane runner2Land of autonomous run A, `internal/core/runner/proc.go` `adminGroupWritableOnly`). Reason: the runner2 re-verification (reverify-runner2) found that on a Homebrew Mac `/opt/homebrew/bin` is `drwxrwsr-x` group admin, so a harness installed there was refused with the `chmod go-w` message; members of the administrator group can already act as root, so that write grants them nothing new, and asking a person to strip Homebrew's own directory mode would break Homebrew.
