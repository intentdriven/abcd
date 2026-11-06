---
schema_version: 1
id: "iss-2611061032580612"
slug: "the-guard-s-script-reading-takes-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "follow-up to the v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Require the holder-kill shape to name kill or a process-listing tool before the rule fires, and add this line to the false-positive corpus."
---

The guard's script reading takes a variable-named program followed by two space-separated flags for kill-by-holder, though the line never mentions kill or lsof: `"$work/record-lint" -root "$top" -slug-names "$names"` in a hook script raised the false-positive sweep to 31 blocks against a ceiling of 30, found while building the slug cap's hook check.
