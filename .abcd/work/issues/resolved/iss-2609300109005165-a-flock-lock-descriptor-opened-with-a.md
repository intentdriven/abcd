---
schema_version: 1
id: "iss-2609300109005165"
slug: "a-flock-lock-descriptor-opened-with-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Open every lock descriptor close-on-exec: add syscall.O_CLOEXEC to openLockFd's open and to the directory-lock primitive's, which the iss-129 consolidation makes the only two raw opens left. Grounds: open(2) on darwin and linux both define O_CLOEXEC as closing the descriptor on execve, and flock(2) states the lock is released only when every descriptor sharing the open file description is closed, so an inherited descriptor keeps the lock; the Go runtime's own os.OpenFile passes O_CLOEXEC for this reason. Proven by a re-exec test whose holder starts a grandchild and is killed: the lock must be granted while the grandchild still runs."
resolution: "Every lock descriptor is close-on-exec: openLockFd opens with O_CLOEXEC (38a442fdc), and WithDirLock, the directory-lock primitive that replaced the decide, intent and spec loops, opens with it too; memory's lock moved onto WithFileLock. A re-exec test kills a holder that left a grandchild running and is granted the lock while the grandchild lives."
impact: fix
resolved_by:
  commit: "38a442fdc"
---

A flock lock descriptor opened with a raw syscall.Open and no O_CLOEXEC is inherited across exec by every child the holder starts, so a child that outlives its dead parent keeps the lock held: the next writer gets contention after its whole budget, although the process that took the lock is gone. Shown by a re-exec test: a WithFileLock holder that starts a sleep and is killed leaves the lock ungrantable for 2s while the sleep lives. The descriptors affected are fsutil.WithFileLock's openLockFd, memory's store lock (writer.go) and the directory locks of decide, intent and spec; os.OpenFile and os.Root add O_CLOEXEC themselves, so WithFileLockIn and history's repoLock are not affected.

## Grounds

- pursued: we expect no lock to outlive the process that took it through a child it started; a re-exec test whose killed holder's grandchild still keeps the lock (contention past the budget) would show it wrong
