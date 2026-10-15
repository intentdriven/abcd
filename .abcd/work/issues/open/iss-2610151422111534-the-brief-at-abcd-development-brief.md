---
schema_version: 1
id: "iss-2610151422111534"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:7 to match the shipped surface: The binary also reads `reservation.github_username` from the machine layer (internal/core/claim/holder.go, holderKey, used by the reserve verb) and `consistency.max_request_bytes` (internal/core/intent/consistency_config.go) through the layered config reader;…"
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:7 says: Beyond those eleven, the binary reads only keys a person opts into: the provider adapter's `oracle` keys and the interview's list style, the implement loop's `pace`, and the runner routes under `roles` and `runner`. The shipped surface differs: The binary also reads `reservation.github_username` from the machine layer (internal/core/claim/holder.go, holderKey, used by the reserve verb) and `consistency.max_request_bytes` (internal/core/intent/consistency_config.go) through the layered config reader; neither key appears anywhere in the chapter. (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a34.)
