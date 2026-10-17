---
schema_version: 1
id: "iss-2610171043574168"
slug: "the-routed-drain-judge-cannot-run-for-a"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain_judge.go"
remedy: "Escape or quote parentheses and commas in the path rules the way the harness's rule syntax allows, or say once on stderr why the judge was not started for this checkout."
---

The routed drain judge cannot run for a checkout or temporary directory whose real path holds a parenthesis or a comma: the path-scoped Read and Edit grants and the checkout denies are refused by the runner's tool-name check, so every move hands the request to the host.
