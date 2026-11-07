---
schema_version: 1
id: "iss-2611070737174057"
slug: "a-stuck-stderr-disables-the-kill-if-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive_signal.go"
remedy: "Cancel and arm the timer before saying the second line, and say the first line in its own goroutine so the second signal is still read; test with a blocked writer and two signals."
---

A stuck stderr disables the kill: if the first line a signal says blocks, the second signal is never read; if the second line blocks, the cancel and the timer never run. The same on main.
