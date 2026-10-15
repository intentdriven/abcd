---
id: dcn-2609300000000024
slug: narrowing-the-administrator-group
date: 2026-09-30
---
2026-09-30 — Narrowing the administrator-group exception in the entry above (lane fix-runnerAdmin of autonomous run A, `internal/core/runner/proc.go` `adminGroupWritableOnly`): a command-line runner admits a harness binary, or a directory it is reached through, that is group-writable (never other-writable) only on darwin and only when the group is gid 80 `admin`; gid 0 is refused on every OS, and gid 80 is refused off darwin. Reason: the entry above granted gid 0 on the premise that members of the administrator group can already act as root, which holds for darwin's admin group (its members may sudo by default) but not for Linux's gid 0 root group nor darwin's gid 0 wheel, whose membership does not by itself let someone act as root (the runner2Land review note). The Homebrew `/opt/homebrew/bin` case the exception exists for is group admin, so it stays admitted.
