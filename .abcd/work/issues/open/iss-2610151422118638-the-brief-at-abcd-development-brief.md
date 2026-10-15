---
schema_version: 1
id: "iss-2610151422118638"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:97 to match the shipped surface: .abcd/config.json carries five top-level blocks: docs, meta, oracle, outbound, repo (jq keys)."
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:97, .abcd/development/brief/05-internals/03-configuration.md says: This repository's own config carries four of these blocks — `docs`, `meta`, `oracle` and `repo`. The shipped surface differs: .abcd/config.json carries five top-level blocks: docs, meta, oracle, outbound, repo (jq keys). (v0.14.2 release crosscheck, stale-count, raised by opus+sonnet in a34.)
