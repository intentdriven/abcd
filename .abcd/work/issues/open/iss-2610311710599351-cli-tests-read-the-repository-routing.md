---
schema_version: 1
id: "iss-2610311710599351"
slug: "cli-tests-read-the-repository-routing"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "make preflight on the README layout branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/disembark_graveyard_test.go"
remedy: "Isolate the cli package's tests from the enclosing checkout's .abcd/config/oracle-routing.json (resolve the repository layer from the test's own temporary root, or pin it empty in the harness), then commit abcd's own routing table, which setup was told to commit on 2026-10-31."
---

Committing a model-routing table to abcd's own repository fails five cli tests. TestDisembarkGraveyardWritesLessons, TestDisembarkGraveyardStdin, TestDisembarkPrinciplesDelegated, TestDisembarkPrinciplesStdin and TestDisembarkPressReleaseDelegated read the checkout's .abcd/config/oracle-routing.json, so the routing notice that no connection serves the economy tier lands in front of the --json document they parse. The tests pass once the file is removed, so they depend on the repository they run in rather than on their own fixtures.
