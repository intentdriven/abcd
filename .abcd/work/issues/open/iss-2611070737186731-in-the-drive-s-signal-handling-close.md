---
schema_version: 1
id: "iss-2611070737186731"
slug: "in-the-drive-s-signal-handling-close"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive_signal.go"
remedy: "Move the stop of the pipe channel to after the wait in close()."
---

In the drive's signal handling, close() stops delivery to the pipe channel before its wait, so a signal's line written to a broken pipe after that ends the process with SIGPIPE before the last words or a way back are printed. The window is tiny and predates the last fix.
