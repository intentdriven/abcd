---
schema_version: 1
id: "iss-129"
slug: "consolidate-bespoke-flock-loops"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "iss-101/102 reviews (2026-07-24 run queue, burst 3)"
found_at: "internal/fsutil/flock.go"
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own (technical, no ruling owed): five bespoke flock sites remain (memory/writer.go, intent/create.go, spec/store.go, decide/decide.go, history/store.go), and three of them lock a directory descriptor, which fsutil.WithFileLock, a lock-file primitive, cannot take; consolidating needs a directory-lock primitive in fsutil and a per-site proof that each refusal and timeout survives, more than an hour on the lock trust path."
---

four bespoke LOCK_EX flock loops remain (memory/writer.go, intent/create.go, spec/store.go, history/store.go) now that fsutil.WithFileLock is the canonical inter-process lock primitive; consolidate them onto it (one-canonical-primitive) — pre-existing, no defect, pure debt