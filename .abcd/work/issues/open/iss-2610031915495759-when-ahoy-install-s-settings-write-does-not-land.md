---
schema_version: 1
id: "iss-2610031915495759"
slug: "when-ahoy-install-s-settings-write-does-not-land"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-2610030814013772 step 2 review 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "Record the length of the change list on entry to stepConfigValues and have the rollback truncate to it, so only this step's echoes are taken back; route every early return through that one rollback (a deferred call keyed on whether the write landed), which is also the fix for the declined-settings retraction the step 2 review found. Pin it with a unit test that preloads a receipt and asserts it survives an unsaved settings change."
---

When ahoy install's settings write does not land, stepConfigValues' rollbackForced (internal/core/ahoy/apply.go) clears the whole change list (a.changes = nil), not only the override echoes stepConfigValues itself added, so a dependency receipt stepDependencies recorded earlier in the same run (a tool installed and verified) is dropped from the result: the person is not told of a tool that was installed.
