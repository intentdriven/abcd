---
schema_version: 1
id: "iss-2608220750024303"
slug: "adr-47-decision-2-s-allowlist-of-generator-added-words-does"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "agent-observation"
found_at: "docs/explanation/rationale.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Amend or supersede adr-47 decision 2 to admit image alt text as a generator-added word?"
remedy: "Waits on the adr-47 decision 2 ruling: if amended: a dated amendment (or a superseding ADR if decision 2's text is false) names image alt text as sourced text, written in the docs page that references the image and never by the generator, proven by a site test that each rendered alt equals its source; if not amended: resolve with the reading that alt text is already a span of a repository file under the single-source rule."
---

adr-47 decision 2's allowlist of generator-added words does not name image alt text, yet the migrated docs pages carry alt text the site inherits; the carve-out wants an explicit ADR amendment

## Remedy grounds (2026-09-29)

SOTA check: WCAG 2.2 technique H37 (https://www.w3.org/WAI/WCAG22/Techniques/html/H37, read 2026-09-29) says alt text must convey the image's meaning in context, and the W3C alt decision tree (https://www.w3.org/WAI/tutorials/images/decision-tree/, read 2026-09-29) gives decorative images an empty alt; both put alt text with the page author, which is where decision 2 already sources it (internal/core/site/assets.go renders the docs page's own alt). Rejected: generated alt text, which decision 2 forbids and H37 would not accept as meaning.
