---
schema_version: 1
id: "iss-2610172134259284"
slug: "the-owed-drain-row-of-the-brief-intent"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief-drift fixes for iss-2610170212055447 and iss-2610170212063646"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Say in the owed-drain row that the drain runs the auditor on its head entry when the auditor's route reaches a provider connection, and that only on the harness route does the plugin page run the loop one audit at a time."
---

The owed-drain row of the brief intent chapter at .abcd/development/brief/04-surfaces/05-intent.md says the drain form of the audit sub-verb runs no reviewer and that with no auditor available every entry stays owed. But when the auditor's route reaches a provider connection the drain sends its head's request there and ingests the verdict (internal/surface/cli/intent_drain.go:66-82), as the same chapter's 'An auditor routed to a provider runs there' passage already says, so the row contradicts both the code and its own chapter.
