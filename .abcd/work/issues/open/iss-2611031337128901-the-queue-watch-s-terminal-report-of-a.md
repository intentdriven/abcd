---
schema_version: 1
id: "iss-2611031337128901"
slug: "the-queue-watch-s-terminal-report-of-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/queue.go"
remedy: "Pass the queue's merge method down from watchQueue and quote the same full command toggleArming runs; a test asserts the quoted command carries the method."
---

The queue watch's terminal report of a dropped pull request quotes a re-arm command gh refuses non-interactively: droppedRefusal in internal/core/implement/loop/queue.go gives 'gh pr merge N --auto --match-head-commit=<sha>' with no merge method, while toggleArming builds the full command (with --merge, --squash or --rebase from the queue's rule) that watchQueue has in hand. Found as a note in #1370's landing review (2026-11-03).
