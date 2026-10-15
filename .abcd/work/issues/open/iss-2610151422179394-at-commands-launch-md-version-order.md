---
schema_version: 1
id: "iss-2610151422179394"
slug: "at-commands-launch-md-version-order"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Correct commands/launch.md (version-order paragraph) so it states what the shipped binary does, as the finding describes."
---

At commands/launch.md (version-order paragraph): The example 'v0.14.2 after v0.15.0' reads oddly in the release that is itself v0.14.2; the rule is correct, only the example is awkward. (v0.14.2 release docs-currency review, raised by sonnet.)
