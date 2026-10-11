---
schema_version: 1
id: "iss-2610110456358519"
slug: "the-surface-register-s-sweep-checks"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Extend the register sweep to agents/: every agents/*.md page has a row, and every agent row names a page, with a test that adding or deleting an agent page without the row fails."
---

The surface register's sweep checks commands/ and hooks/ pages against the register in both directions, but the prompt files under agents/, which a harness registers as invocable agents, are checked in neither: nothing asserts an agent page has a register row, and nothing notices one added or removed.
