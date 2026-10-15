---
schema_version: 1
id: "iss-2610141705124287"
slug: "the-brief-surface-index-row-for"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Point the docs-currency-reviewer row at launch (the release gate) instead of docs."
---

The brief surface index row for docs-currency-reviewer says it serves docs, but neither commands/docs.md nor abcd docs (cite, fidelity) names or takes that agent output; it runs as a release-gate pass under launch (commands/launch.md:898-901), as 10-docs.md:220 says.
