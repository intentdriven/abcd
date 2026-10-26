---
schema_version: 1
id: "iss-2610241414129778"
slug: "abcd-capture-accepts-a-found-at-path"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture"
remedy: "Check --found-at against the tree at capture time; refuse a path that does not exist (exit 2, naming the path and the closest existing one), and accept a location described in words only through an explicit flag such as --found-at-text, so a typo or a not-yet-written file is caught when it is made."
---

abcd capture accepts a --found-at path that does not exist in the tree without saying so: on 2026-10-24, during the setup redesign build, a capture naming a file that had not been written yet produced a record whose location points nowhere, and the mistake was noticed only later by hand.
