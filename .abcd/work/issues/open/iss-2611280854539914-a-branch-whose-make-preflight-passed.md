---
schema_version: 1
id: "iss-2611280854539914"
slug: "a-branch-whose-make-preflight-passed"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "surge 25-30/11, model matrix PR-3 push"
origin: researcher-authored
production_mode: hand-written
found_at: ".gitleaks.toml"
remedy: "Add provider key shapes the forge's push protection flags (an OpenRouter key among them) to .gitleaks.toml as custom rules so secrets-check refuses them at the change, and give test fixtures a convention of building a key-shaped value at run time (prefix plus strings.Repeat) rather than committing the literal; name that convention where the test style is documented."
---

A branch whose make preflight passed, secrets-check included, was refused at push by the forge's secret scanning: a test fixture held an OpenRouter-shaped placeholder key (the provider prefix and 64 zeros) that the gitleaks default rules and the repository's .gitleaks.toml pass. The receipt promises the tree CI checks out, but the push itself is gated by a stricter scanner, so the failure surfaces only after a full preflight, and the history the guard will not let anyone rewrite then has to be re-cut as a new branch.
