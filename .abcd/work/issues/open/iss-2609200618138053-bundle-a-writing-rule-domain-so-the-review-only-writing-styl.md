---
schema_version: 1
id: "iss-2609200618138053"
slug: "bundle-a-writing-rule-domain-so-the-review-only-writing-styl"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "teachingprep-2026-27-rewrite-handover"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/writing-style.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Bundle a WRITING rule domain with a broad recall list, decided with the stop-word policy?"
---

Bundle a WRITING rule domain so the review-only writing-style rules reach agents in every managed repo. docs/reference/writing-style.md holds the canonical rules, but none of the eight bundled domains (COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS, LIFEBOAT, PII, OPINIONS) carries them, so an agent in a managed repo never sees them unless the repo declares a custom domain. The four review-only rules (capital after a colon, lower case after a semicolon, serial comma, no em dash inside a list item) are exactly the ones no lint catches (adr-54), so injection is the only enforcement they can have; the observed effect in TeachingPrep is repeated lowercase-after-colon in agent-written handover files and ledger text from two different agent sessions. TeachingPrep has added a repo-override WRITING domain in .abcd/rules.json as the interim; a bundled default with the same rules and a broad recall list (write, draft, prose, note, handover, readme, markdown, document, report, message, edit, style) would replace it, and the repo override should then be removed. Relation to iss-2609012039220931: a bundled domain with common-word recall is exactly the case where recall breadth is a design choice rather than an abuse, so decide the two together.
