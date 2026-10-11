---
schema_version: 1
id: "iss-2609200618138053"
slug: "bundle-a-writing-rule-domain-so-the"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "teachingprep-2026-27-rewrite-handover"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/writing-style.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed I): Bundle a WRITING rule domain with a broad recall list, decided with the stop-word policy?"
remedy: "Waits on ruling I (bundle a WRITING domain, decided with the stop-word recall policy of iss-2609012039220931): if bundled, add a WRITING domain to internal/core/rules/defaults/rules.json carrying the four review-only rules of docs/reference/writing-style.md and a recall list of prose-task words, proven by a loader test that a handover-writing prompt injects it and a code-only prompt does not; if not, document the repository-override WRITING domain as the supported route and resolve this record as wontfix."
---

Bundle a WRITING rule domain so the review-only writing-style rules reach agents in every managed repo. docs/reference/writing-style.md holds the canonical rules, but none of the eight bundled domains (COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS, LIFEBOAT, PII, OPINIONS) carries them, so an agent in a managed repo never sees them unless the repo declares a custom domain. The four review-only rules (capital after a colon, lower case after a semicolon, serial comma, no em dash inside a list item) are exactly the ones no lint catches (adr-54), so injection is the only enforcement they can have; the observed effect in TeachingPrep is repeated lowercase-after-colon in agent-written handover files and ledger text from two different agent sessions. TeachingPrep has added a repo-override WRITING domain in .abcd/rules.json as the interim; a bundled default with the same rules and a broad recall list (write, draft, prose, note, handover, readme, markdown, document, report, message, edit, style) would replace it, and the repo override should then be removed. Relation to iss-2609012039220931: a bundled domain with common-word recall is exactly the case where recall breadth is a design choice rather than an abuse, so decide the two together.

## Remedy grounds (2026-09-29)

- adr-54 keeps these four rules out of the lints, so injection is the only enforcement they can have; the open choice is recall breadth, which is local policy, and no outside practice was consulted.
- Rejected: a lint for the four rules, which adr-54 rules out.
