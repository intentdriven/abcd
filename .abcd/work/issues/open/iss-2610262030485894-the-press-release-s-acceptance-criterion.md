---
schema_version: 1
id: "iss-2610262030485894"
slug: "the-press-release-s-acceptance-criterion"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 47 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:42"
remedy: "Rewrite the quoted-create acceptance criterion (line 42): running the quoted create writes a draft whose Press Release is the quoted text, with Why This Matters and Acceptance Criteria seeded as prompts for the human, no interview and no persona-registry quote."
finding_ref: cfk-79f12c9666b28c66
---

The press release's acceptance criterion expects an interview yielding a persona quote and Given-When-Then criteria; the intent chapter's create seeds the criteria as prompts

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:42` — "file exists with press-release content, customer quote from the persona registry, and a `## Acceptance Criteria` section in Given-When-Then format"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:318` — "with Why This Matters and Acceptance Criteria seeded as prompts for the human to fill"

01-press-release's criterion says that when the create's interview completes the draft holds press-release content, a customer quote from the persona registry and Given-When-Then criteria, while 05-intent says the quoted create runs no interview and seeds Why This Matters and Acceptance Criteria as prompts for the human to fill. Audited against the shipped create, that criterion cannot read MET.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 47.
