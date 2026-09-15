---
schema_version: 1
id: "iss-2609012037143368"
slug: "the-capture-package-s-ledger-writes-are-absolute-path-operat"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/alloc.go"
---

The capture package's ledger writes are absolute-path operations — createPlaceholder through syscall.Open, the .iss-alloc.lock flock path, fsutil.WriteFileAtomicPreserveMode for the commit and the stamps, os.Mkdir for the directories — each guarded by its own Lstat or O_NOFOLLOW check. The GHSA-865x fix adds a per-segment ancestor walk, which closes the committed-symlink case but leaves the lstat-to-mkdir window a local racer could use, the same window memory.memoryDir accepts. The durable closure is os.Root: fsutil already carries CreateExclusiveIn, WriteFileAtomicInRoot and ReadGuardedInRoot for exactly this class, and a store opened once as a root cannot be redirected between a check and a write. Converting the capture store to it is a package-wide refactor of every write site and the lock path, not a one-site fix, and it is the shape a structural record-store containment rule (iss-2608301308367566) would want every store to take. Recorded as the follow-up the advisory fix does not attempt.
