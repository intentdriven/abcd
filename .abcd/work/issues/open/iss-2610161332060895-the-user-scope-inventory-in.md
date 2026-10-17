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
---

The user-scope inventory in 03-configuration.md omits interviews/ (setup answers that change the machine) and backups/ (harness settings copies the status line takes), which 01-ahoy's tree lists, though the chapter says the two are one list and must agree.
