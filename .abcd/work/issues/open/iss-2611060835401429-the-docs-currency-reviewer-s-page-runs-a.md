---
schema_version: 1
id: "iss-2611060835401429"
slug: "the-docs-currency-reviewer-s-page-runs-a"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/docs-currency-reviewer.md"
remedy: "Have the page name how to run the check from a checkout, `abcd lint docs` if that is the same check, or drop the step."
---

The docs-currency reviewer's page runs a `docs-currency-lint` command first when it is on PATH (`agents/docs-currency-reviewer.md:23`), but no build produces that command, so the pass is skipped and only the reviewer's reply says so. A reader of the page believes a deterministic check stands behind the review.
