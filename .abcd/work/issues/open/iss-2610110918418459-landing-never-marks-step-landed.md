---
schema_version: 1
id: "iss-2610110918418459"
slug: "landing-never-marks-step-landed"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "fidelity audit of itd-2609212103565953, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "On each lane's landing, before the push, add the '- landed: <pull request or commit>' line under the lane's spec step in the open spec on the lane's branch (the format spec/steps.go parses), committed with the landing's records; test that a three-step run whose lane 2 is handed back leaves steps 1 marked and a --remainder close carries only steps 2 and 3."
---

The implement loop's landing never writes a spec step's '- landed:' line, so a spec the loop landed part of still reads those steps as unlanded: a later 'spec close --remainder' carries steps that already merged into the remainder, and a fresh 'abcd build' of the same spec opens a lane for them again. Found by the fidelity audit of itd-2609212103565953 (receipt rcp-2257cc3a38d9, criterion ac-3 MET_WITH_CONCERNS); spc-2609202134338445's Progress already says the landing does not mark a non-closing lane's step.
