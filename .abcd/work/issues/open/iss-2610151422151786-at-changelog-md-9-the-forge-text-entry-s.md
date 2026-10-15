---
schema_version: 1
id: "iss-2610151422151786"
slug: "at-changelog-md-9-the-forge-text-entry-s"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Correct CHANGELOG.md:9 so it states what the shipped binary does, as the finding describes."
---

At CHANGELOG.md:9: The forge-text entry's list of what the check does not cover leaves out the pr and issue close --comment forms, a body or path the line does not fix (`--body "$BODY"`), and a short flag bundled behind another (`-db x`). commands/guard.md lists all of these. Verified: a pr close --comment carrying a tool footer is an allow, while the same text on pr comment --body is blocked. A reader of the release notes could take closing comments to be covered. (v0.14.2 release docs-currency review, raised by opus.)
