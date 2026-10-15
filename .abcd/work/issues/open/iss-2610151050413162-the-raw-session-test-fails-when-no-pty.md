---
schema_version: 1
id: "iss-2610151050413162"
slug: "the-raw-session-test-fails-when-no-pty"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/term"
remedy: "Skip TestRawSessionLeavesNothingBehind (and any sibling that opens /dev/ptmx) with a stated reason when the pseudo-terminal cannot be opened, as the ptytest helpers do, instead of failing the package."
---

TestRawSessionLeavesNothingBehind fails, rather than skips, when the machine cannot open a pseudo-terminal: a preflight on 2026-10-15 failed only on 'opening a pseudo-terminal: /dev/ptmx: device not configured' with 40 of 511 ptys in use, and the same test passed three times straight after. A transient pty failure therefore fails a whole preflight.
