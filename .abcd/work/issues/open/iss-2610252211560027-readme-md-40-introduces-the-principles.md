---
schema_version: 1
id: "iss-2610252211560027"
slug: "readme-md-40-introduces-the-principles"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D8"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Record the three as principle files under .abcd/development/principles/, in the product thinker's wording, so the linked page lists every principle the README names. If they are not to be recorded, take the three bullets out of README.md:47, :57 and :60 rather than present them as principles of the record."
---

README.md:40 introduces the principles with a link to the record site's foundations page ('abcd is founded on several [principles](https://abcdev.app/record/foundations/), ...'). README.md:47, :57 and :60 then name three principles that page cannot carry: '**abcd builds abcd**', '**Work starts from an intent**' and '**The record is part of the product**'. The foundations page is built from the principle store alone (internal/core/site/foundations.go:16-18, explorer.go:721-746). .abcd/development/principles/ has no record for any of the three, and git grep finds those names only in README.md. The other five are in the store: prefer-the-experiment-to-the-inference.md, verifier-selects-gates-decide.md, less-but-better.md, the-record-lands-with-the-act.md and enforcement-claims-are-facts.md. So a reader who follows the link finds five of the eight principles the README names.
