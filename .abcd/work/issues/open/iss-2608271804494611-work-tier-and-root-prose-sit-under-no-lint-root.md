---
schema_version: 1
id: "iss-2608271804494611"
slug: "work-tier-and-root-prose-sit-under-no-lint-root"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/record-lint.json"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: lint-root coverage for the working tier and root prose, or a record of why they stay ungated). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

everything under .abcd/work/ sits under no lint root: record-lint's root is .abcd/development and docs-lint's roots are docs and README.md, so the issue ledger, the reviews charter, rulesets/, DECISIONS.md, the .abcd root config files, the plugin trees, and every root prose file except README.md are gated by nothing — which means structural fixes in those areas can silently regress. iss-279 and iss-2608230752354928 capture two slices (docs-lint roots; relative links under work/); this record is the umbrella: decide the lint-root coverage for the working tier and root prose, or record why they stay ungated.