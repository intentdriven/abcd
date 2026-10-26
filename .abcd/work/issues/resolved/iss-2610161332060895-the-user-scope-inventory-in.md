---
schema_version: 1
id: "iss-2610161332060895"
slug: "the-user-scope-inventory-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "deferred config-brief fixes, inventory check"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add interviews/ and backups/ to the 03-configuration.md user-scope inventory."
resolution: "a394eb10a added interviews/ and backups/ to the user-scope inventory in 03-configuration.md, which is what this record asked for."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The user-scope inventory in 03-configuration.md omits interviews/ (setup answers that change the machine) and backups/ (harness settings copies the status line takes), which 01-ahoy's tree lists, though the chapter says the two are one list and must agree.

## Grounds

- pursued: we expect the configuration chapter's inventory to name interviews/ and backups/ from now on; shown wrong if a later edit drops either while 01-ahoy's tree still lists it. The remaining omissions are held by iss-2610161332064271.
