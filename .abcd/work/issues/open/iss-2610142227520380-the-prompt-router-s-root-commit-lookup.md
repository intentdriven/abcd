---
schema_version: 1
id: "iss-2610142227520380"
slug: "the-prompt-router-s-root-commit-lookup"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "spc-2610111627213138 security round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/rules_prompt.go"
remedy: "Cache the root commit per checkout (it never changes), or read it from the commit-graph, so the 2 s cap is never reached after the first lookup."
---

The prompt router's root-commit lookup (boundedRootCommit) is a full-history rev-list under a 2 s cap. On a very large repository without a commit-graph it could exceed that cap on every prompt, so the live drain and the tied-note recall would skip every prompt, each with a stderr line, and staged transcripts would never drain there. It measured 0.01 s on this repository (7,873 commits), so the failure is projected, not observed.
