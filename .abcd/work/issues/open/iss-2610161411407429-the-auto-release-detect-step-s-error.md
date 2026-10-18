---
schema_version: 1
id: "iss-2610161411407429"
slug: "the-auto-release-detect-step-s-error"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "security review of the release-chain fix"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows"
remedy: "Add the next step to the message: what to check and who to ask when a re-run fails identically."
---

The auto-release detect step's error message does not say what to do when re-running the release fails the same way again.
