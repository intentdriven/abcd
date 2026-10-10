---
schema_version: 1
id: "iss-2610091903077581"
slug: "abcd-report-lets-an-agent-in-an-abcd-managed-repository-file"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "2026-10-09 verb-split interview"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/report.md"
remedy: "Have the report page and verb require the person's explicit yes before a report is written from a managed repository (a question through the host's question tool showing the report's title and text, or a --confirmed flag the agent may pass only after that yes), refuse otherwise, and test that a report without the confirmation writes nothing."
refines: [iss-2610031236155833]
---

abcd report lets an agent in an abcd-managed repository file a defect report or proposal about abcd into the abcd inbox without asking anyone: commands/report.md says nothing about permission. The product thinker ruled on 2026-10-09 that report stays an agent's command but must ask the person's permission first when it runs in an abcd-managed repository, since the report carries that project's words to another project's inbox.
