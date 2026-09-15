---
schema_version: 1
id: "iss-2609020716579024"
slug: "a-resolution-note-that-names-the-tests-proving-a-limb-is-tru"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues"
---

A resolution note that names the tests proving a limb is trusted by every later reader, and one was false: the transcript lane's note named three tests for the drain-side removal guard and none of them exercised it (the ruthless review reverted the guard and the suite stayed green). A reviewer or a lint could check the claim mechanically: every TestX a resolution names must exist, and a mutation of the described limb should fail at least one of them. The cheap rung is record-lint verifying that each named test exists; the mutation check is a review discipline until it earns a tool.
