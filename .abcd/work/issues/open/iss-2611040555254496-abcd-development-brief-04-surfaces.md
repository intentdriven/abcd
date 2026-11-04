---
schema_version: 1
id: "iss-2611040555254496"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Limit the sentence to the check and refer to the fail-open-loud section for the hook."
---

.abcd/development/brief/04-surfaces/17-guard.md:508-509 says `.abcd/guard.json` must declare schema_version 1 or the guard refuses to run at all. That holds for the check, exit 2. On the hook the repository's overrides are dropped with a notice and the bundled hazards still decide, so a force push is still denied, as .abcd/development/brief/04-surfaces/17-guard.md:492-496 describes.
