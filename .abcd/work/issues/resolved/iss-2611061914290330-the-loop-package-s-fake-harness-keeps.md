---
schema_version: 1
id: "iss-2611061914290330"
slug: "the-loop-package-s-fake-harness-keeps"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal_test.go"
remedy: "Write the count atomically, a temporary file in the same directory and then a rename, so a kill leaves the old count or the new one, and hold that with a test whose write is cut off after its open."
resolution: "The fake's count goes through an atomic write, so a kill leaves the old count or the new one, and TestACounterWriteCutOffKeepsTheScriptsPlace holds it: that test failed before the change with an empty count. The CI failure was not reproduced, so this removes the one way found for a launch to lose its place and does not show the flake is gone."
impact: internal
resolved_by:
  commit: "0e350bf4d"
---

The loop package's fake harness keeps its place in a test's script by reading a count from a file and rewriting it with os.WriteFile, which truncates and then writes, in a process a 200 ms hang limit kills. A write cut off between the two leaves the count empty, and the next launch reads zero and takes the script's first line instead of its own. TestASilentLaunchCountsOnlyIfAProbeAnswers failed in CI (run 42271547732, job 128077204775) on a script whose only line that writes a receipt is its first and whose judge had already taken it: eight implementer launches each failed with nothing to show what failed them, the last reported killed at the limit, then a receipt stood and verified with no further launch, and the lane finished. That fits the count having gone back to zero, but the log does not show the count, and the failure was not reproduced.
