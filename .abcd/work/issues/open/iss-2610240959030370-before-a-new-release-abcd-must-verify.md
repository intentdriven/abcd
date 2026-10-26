---
schema_version: 1
id: "iss-2610240959030370"
slug: "before-a-new-release-abcd-must-verify"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
remedy: "The release cut tries each dependency's and each managed-repository pin's newest release against abcd's gates, moves the pin when they pass, and otherwise holds the last passing version and records why."
---

before a new release, abcd must verify that all dependencies and especially those it pins to an abcd-managed repository are moved to the latest, verified version -- verified meaning that they are guaranteed to work with abcd. It should stay with a previous version if the newest one breaks abcd.
