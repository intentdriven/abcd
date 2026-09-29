---
schema_version: 1
id: "iss-2609291246244867"
slug: "the-history-registry-s-locked-rewrite-took-its-lock-through"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
---

The history registry's locked rewrite took its lock through one walk of ~/.abcd/history and loaded and wrote index.json through two further walks (registerRepo: withHistoryLock, loadHistoryIndex, writeHistoryIndex each opened the directory afresh), so a same-uid swap of ~/.abcd/history once the lock was held sent the load and the write to a directory whose lock was never taken. The same shape as iss-2609290300313698, found while sweeping that pattern.
