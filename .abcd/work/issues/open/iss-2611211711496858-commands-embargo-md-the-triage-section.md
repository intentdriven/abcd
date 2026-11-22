---
schema_version: 1
id: "iss-2611211711496858"
slug: "commands-embargo-md-the-triage-section"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "lab-261121150749-bb30bb5"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/embargo.md"
remedy: "Route the embargo triage verdict to the technical facilitator rather than the product thinker: commands/embargo.md asks for a Product-chip confirmation, though the verdict is a technical judgement, and docs/explanation/roles.md gives technical judgements to the technical facilitator"
---

commands/embargo.md (the Triage section) tells the agent to ask the product thinker to confirm a triage verdict under a Product chip. The verdict is a judgement about whether a report is a vulnerability, which roles.md gives to the technical facilitator. On 21 Nov 2026 the person, as product thinker, declined four such questions as ones they could not decide and delegated them to the agent sessions. Lab lab-261121150749-bb30bb5 (F-2) traced the routing to this page.
