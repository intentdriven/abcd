---
schema_version: 1
id: "iss-2610151422141736"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite .abcd/development/brief/02-constraints/04-naming.md:101 to match the shipped surface: Bare `abcd reserve` also refuses as a usage error (`abcd reserve --help`: Usage `abcd reserve <record-id>`), and bare `abcd report` refuses anywhere but a terminal. The enumeration this paragraph defers to (04-surfaces/README.md § Bare invocation) names…"
---

The brief at .abcd/development/brief/02-constraints/04-naming.md:101 says: "three verbs (`launch`, `decide` and `build`) refuse instead of rendering" on a bare call. The shipped surface differs: Bare `abcd reserve` also refuses as a usage error (`abcd reserve --help`: Usage `abcd reserve <record-id>`), and bare `abcd report` refuses anywhere but a terminal. The enumeration this paragraph defers to (04-surfaces/README.md § Bare invocation) names reserve, so the count of three is stale. (v0.14.2 release crosscheck, stale-count, raised by opus in b00.)
