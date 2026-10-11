---
schema_version: 1
id: "iss-2610112201576037"
slug: "teststopleavesnothinglistening-is-flaky"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "merge-queue run of #935, reported by a peer session"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/proc_linux.go"
remedy: "In internal/surface/dashboard/proc_linux.go, map syscall.ESRCH from both the stat read and the exe readlink to errProcessGone, beside fs.ErrNotExist. Add a test that feeds readProcess an ESRCH through a seam. Sweep the darwin reader and any other /proc reader for the same gap."
---

TestStopLeavesNothingListening is flaky on Linux and twice dropped a merge-queue entry (the run that landed #935). The dashboard's Linux readProcess reads /proc/<pid>/stat and /proc/<pid>/exe and treats only ENOENT, a zombie or a dead state as a gone process. A read that races the server's exit fails with ESRCH ("no such process") while the /proc entry still exists, so sameProcess returns that error and Stop fails, where it should have seen the process gone.
