---
schema_version: 1
id: "iss-2608270500198764"
slug: "record-lint-validateintent"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/record-lint"
resolution: "record-lint refuses every intent id intent.Load refuses via the shared recordid.ValidIntentID"
impact: fix
resolved_by:
  commit: "bf9c25b4"
---

record-lint validateIntent/checkRecordFilename never validate the intent frontmatter id, while intent.Load fail-closes the whole corpus on an empty id, so a malformed id passes lint but breaks the loader. GitHub mirror: #279