---
schema_version: 1
id: "iss-2611061656474206"
slug: "commands-drain-md-fences-the-two-login"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "review of the drive's profiles"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Move all three lines to prose that tells the person to type them, as the page's other person-only steps do."
decision_needed: "Should the line that makes the two profile directories and the two login lines on commands/drain.md stop being fenced steps an agent runs verbatim and become prose steps only the person types, given that this changes what the agent is allowed to run around login credentials and also rewords a page a person reads and follows?"
---

commands/drain.md fences the line that makes the two profile directories and the two login lines that use them, though a fenced line on a command page is what an agent runs verbatim and these are the person's own steps.
