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
remedy: "Have ahoy install offer, per repository, the choice dcn-2611012102280663 rules: the subscription (the shipped default) or the API key for every agent abcd hands out. Explain in one plain paragraph what each bills, write the choice to the person's git-ignored per-repository local layer only on their yes (never to a committed file, which stays refused a runner route), name where the key is read from, and have ahoy report the route in force and whether a key is reachable, so a person never hand-edits a roles block or wonders what it does."
---

Routing abcd's agent roles to the person's API key takes a hand-written roles block in ~/.abcd.noindex/config.json (roles.<role>.runner: "claude" for the implementer, both reviewers, the intent auditor, the drain judge and the planning interviewer) plus an API key the runner can reach, and nothing in abcd offers, explains or checks either: on 2026-11-01 the person pasted the block as given with no way to tell what it was for, and the key had to be passed by hand on each command from the login keychain. A managed repository's person should get this route by default once they have an API key, without editing JSON.
