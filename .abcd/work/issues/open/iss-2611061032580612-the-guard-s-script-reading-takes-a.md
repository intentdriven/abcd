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
found_at: "internal/core/guard/unknown.go"
remedy: "Narrow the reading of a program named by a variable so a script naming none of kill, lsof or fuser does not draw kill-by-holder, and add the hook's lines to the false-positive corpus."
---

The guard's reading of a program named by a variable (`program-name-unknown`) blocks a hook script as kill-by-holder, though the script never mentions kill, lsof or fuser. Adding `"$work/record-lint" -root "$top" -slug-names "$names"` to the commit-msg hook, with earlier lines writing `$names` from `git diff`, made the file read whole draw that block and raised the false-positive sweep to 31 blocks against a ceiling of 30 (`TestFalsePositiveSweepCeiling`), found while building the slug cap's hook check. The kill-by-holder entry's own pattern still needs kill with lsof or fuser, so the firing comes from the reading that stands in for it. The sweep passes on a tree without that line.
