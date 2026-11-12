---
schema_version: 1
id: "iss-2611120957377169"
slug: "a-drive-that-takes-up-one-of-its-lanes"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Pass that lane over in the drive's own memory, never in the run's file or control record: the next move skips it, the drive says once what it left and why, the drain does not stop at its cap or as empty while such a lane stays in progress, a later drive takes it up again through the usual claim and judge, and a drive whose only work left is passed-over lanes ends with an outcome of its own"
resolution: "A drive that takes up a lane whose run's file states another schema version it may not save now passes that lane over in its own memory and goes on with its other lanes; a drive left with only such lanes ends on its own outcome, naming them, with the drain left open for a later drive where it would have stopped at its cap or as empty"
impact: fix
---

A drive that takes up one of its lanes whose run file states another schema version, and that this session may not save, ends the whole drive there, because dropping the lane needs a save of that run the judge refuses and going on would send the next move straight back into the same lane; an unattended drive with other lanes and issues left then stops on that one lane
