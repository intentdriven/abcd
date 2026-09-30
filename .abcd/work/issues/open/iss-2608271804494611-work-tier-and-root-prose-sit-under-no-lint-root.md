---
schema_version: 1
id: "iss-2608271804494611"
slug: "work-tier-and-root-prose-sit-under-no-lint-root"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/record-lint.json"
remedy: "Waits on ruling F: if extended, add AGENTS.md, CLAUDE.md, GEMINI.md, ACKNOWLEDGEMENTS.md and RELEASE.md as file entries to the links_resolve and harness_leak `extra_roots` in `.abcd/record-lint.json`, fixing what they surface in the same change, proven by a lint test that a broken relative link planted in a root prose file is refused; if kept ungated, name each ungated file and its reason in `.abcd/development/brief/05-internals/06-lint.md` and resolve this record with that line."
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: the working tier is partly gated since the record. links_resolve and harness_leak reach .abcd/work through extra_roots, record_schema, lint-issues and issue-drift read the ledger, lint-decisions reads DECISIONS.md, lint-reviews the reviews charter, the two context rules CONTEXT.md, and changelog_unreleased_empty the changelog. Still under no root: the root prose other than README.md (AGENTS.md, CLAUDE.md, GEMINI.md, ACKNOWLEDGEMENTS.md, RELEASE.md) and the .abcd root config files. Extend the roots or record why they stay ungated. The slice this record names as iss-2608230752354928 is a history-store record and does not match the description given for it. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

everything under .abcd/work/ sits under no lint root: record-lint's root is .abcd/development and docs-lint's roots are docs and README.md, so the issue ledger, the reviews charter, rulesets/, DECISIONS.md, the .abcd root config files, the plugin trees, and every root prose file except README.md are gated by nothing — which means structural fixes in those areas can silently regress. iss-279 and iss-2608230752354928 capture two slices (docs-lint roots; relative links under work/); this record is the umbrella: decide the lint-root coverage for the working tier and root prose, or record why they stay ungated.

## Remedy grounds (2026-09-29)

- Both rules already take `.abcd/work` through `extra_roots`, and the lint chapter says an entry may be one file, so extending coverage is configuration plus the findings it surfaces rather than new code.
- No outside-practice check: the question is this repository's own gate coverage.
- Rejected: widening docs-lint's roots to the root prose, whose host-agnostic rules are written for user-facing docs and would need escapes across the credit file and the host bridge files.
