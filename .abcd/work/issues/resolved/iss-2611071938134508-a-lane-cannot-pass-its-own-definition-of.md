---
schema_version: 1
id: "iss-2611071938134508"
slug: "a-lane-cannot-pass-its-own-definition-of"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first live abcd drain --drive"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/main_test.go"
remedy: "State a lane's definition of done in AGENTS.md as make precheck and the changed packages' race tests through the narrow gate, each inside the limit, since the drive runs make preflight itself before landing; and have the CLI tests' TestMain clear the stage's launch names, so no fixture reads them."
resolution: "AGENTS.md states a lane's shorter definition of done, inside the harness's command limit, and the CLI tests' TestMain clears the stage's launch names."
impact: fix
---

A drive's lane could not pass its own definition of done. The harness a stage runs ends each command after about ten minutes, which a whole make preflight outlasts, so every implementer's preflight was terminated partway and its receipt refused. And the stage's launch names the lane's base commit, declared paths and writable folders in the environment, which the CLI package's tests inherit, so 54 of them failed in a lane, read against fixture repositories where that commit and those paths are not theirs.
