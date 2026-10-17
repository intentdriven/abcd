---
schema_version: 1
id: "iss-2610170730017996"
slug: "testmarkingafloodshownfitstheprompt-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "building iss-2610161645224876 (announce before abcd's own pushes)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_test.go"
remedy: "Make the flood test's time bound hold under load, for instance by measuring the work against a bound scaled to a calibration read made in the same test, or by moving the wall-clock assertion behind the load-check tag so a loaded machine reports it rather than failing the package."
resolution: "The status tests stretch the prompt-read budget and their wall-clock bounds by a fixed tenfold slack from the shipped value and log the measured time (internal/core/claim/wallclock_test.go)."
impact: fix
resolved_by:
  commit: "ae938359f87b90df1ec590f6fe4ab864d28b64aa"
---

TestMarkingAFloodShownFitsThePrompt in internal/core/claim fails on a loaded machine: at a load average of 15 to 20 the read and marking of 300 announcements took 281 to 536 ms, past its 500 ms bound or the prompt's deadline. It fails the same way on an untouched copy of origin/main at 998883ef0, so it is a wall-clock bound that depends on the machine's load, not a defect the code under test introduced.
