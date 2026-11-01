---
schema_version: 1
id: "iss-2611012054262885"
slug: "routing-abcd-s-agent-roles-to-the-person"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "routing build roles to the API key, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "Have ahoy install offer the runner route as the default for an abcd-managed repository, as dcn-2611012102280663 rules: explain in one plain paragraph that handed-out agents then bill the person's API key rather than their subscription, write the roles block to the machine layer on their yes, name where the key is read from, and have ahoy report the route and whether a key is reachable. Whether a subscription route is also offered stays the open question iss-2610230716454430."
---

Routing abcd's agent roles to the person's API key takes a hand-written roles block in ~/.abcd.noindex/config.json (roles.<role>.runner: "claude" for the implementer, both reviewers, the intent auditor, the drain judge and the planning interviewer) plus an API key the runner can reach, and nothing in abcd offers, explains or checks either: on 2026-11-01 the person pasted the block as given with no way to tell what it was for, and the key had to be passed by hand on each command from the login keychain. A managed repository's person should get this route by default once they have an API key, without editing JSON.
