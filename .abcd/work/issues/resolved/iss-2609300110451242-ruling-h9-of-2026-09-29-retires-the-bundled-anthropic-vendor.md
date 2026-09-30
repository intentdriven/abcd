---
schema_version: 1
id: "iss-2609300110451242"
slug: "ruling-h9-of-2026-09-29-retires-the-bundled-anthropic-vendor"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/config.go"
remedy: "Amend AC3 and the spec to the allowlist-alone reading with an Audit Notes line naming H9 and the old wording, supersede adr-2609221009491186 decision 2 with a new ADR, and remove the bundled list from internal/core/oracle; grounds: ruling H9 (2026-09-29) and its H10 recording shape; shown wrong if any surface still names a bundled entry"
resolution: "Ruling H9 is applied: the bundled anthropic/* denylist is gone from internal/core/oracle, the provider's allowlist alone decides, itd-2609081951381895 criterion 3 and spc-2609221011153746 read the allowlist alone with the old wording in the intent's Audit Notes, and adr-2609300107513982 supersedes adr-2609221009491186"
impact: additive
resolved_by:
  commit: "628db59d4"
---

Ruling H9 of 2026-09-29 retires the bundled anthropic/* vendor denylist, which changes the reading of itd-2609081951381895 acceptance criterion 3 (a listed model whose prefix matches the vendor denylist is refused at read, and no allowlist entry overrides it) and of adr-2609221009491186 decision 2: with no bundled list the provider's allowlist alone decides, so a listed model of any vendor is served, and only an oracle.denylist entry the repository or the machine writes refuses a listed model

## Grounds

- pursued: a listed model of any vendor now loads, is admitted and is called (TestTheAllowlistAloneDecides, TestCallServesAListedModelOfAnyVendor), and no surface names a bundled entry (TestAhoyProvidersExplainsWithNothingConfigured); shown wrong if any read, setup or call still refuses a listed model that no oracle.denylist entry names
