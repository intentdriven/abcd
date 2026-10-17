---
schema_version: 1
id: "iss-2610162306239284"
slug: "the-load-gate-s-stand-back-window-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "re-check review of fix/load-queue-start-order 45bb56545 (abcd-05, Opus)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "Measure the window from the waiter's own re-check read (or drop the age filter, since the window can only discard real openers), and add a test that a too-wide window fails."
---

The load gate's stand-back window (the open pause plus the start tolerance) is measured from the look, not from the waiter's own re-check read, so a waiter whose path from that read to starting its mark is slow under load (about 2.5 s) discards a concurrent lower-pid opener it did see, and both run their commands, which can bring back two preflights each waiting on the other's running waiter; no test pins the window's bound.
