---
schema_version: 1
id: "iss-2611101320494180"
slug: "abcd-embargo-has-no-form-for-a-finding"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/intake.go"
remedy: "Waits on a ruling first: then add an embargo add form for an in-development finding, with a not-yet-reachable state and a fix-by step, that drafts the private advisory itself."
---

abcd embargo has no form for a finding raised during development, before any release carries the code: it takes in advisories that already exist on the forge, so the draft is made with the forge's own commands first. The version range such a draft carries then names releases in which the code is not reachable, which would read wrongly if it were ever published. Seen on 8 Nov at 11:55. A person must choose whether abcd adds an embargo entry for an in-development finding, and what state and fix-by step it carries.
