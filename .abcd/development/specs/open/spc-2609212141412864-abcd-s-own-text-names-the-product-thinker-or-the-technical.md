---
id: spc-2609212141412864
slug: abcd-s-own-text-names-the-product-thinker-or-the-technical
intent: itd-2609212137129937
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-own-text-names-the-product-thinker-or-the-technical

## Summary

The design record for itd-2609212137129937: the sweep, the lint, the question templates and the glossary entries.

## Scope

1. **The lint first**: `maintainer` added to the docs-lint banned tokens with roots widened to `commands/`, `.abcd/rules.json` and the bundled rules source; escapes marked with the existing allow comment (criterion 2).
2. **The sweep**: forty-plus rewrites, each to the role the sentence meant, in one reviewed change; the persona hint becomes "open-source project lead" (criterion 1).
3. **The templates**: the plugin pages' question blocks name the addressee from the mode (criterion 3).
4. **The glossary**: `product-thinker.md`, `technical-facilitator.md` (criterion 4).
5. **The test** over rendered help and pages (criterion 5).

## Out of scope

- A third role; role responsibilities.

## Approach

Lint before sweep so the sweep is watched red then green; the rewrite table is reviewed by the record-discipline reviewer since each sentence's meaning decides the role.

## Footprint

- packages: internal/core/lint (docs), commands/, .abcd/development/brief, .abcd/development/principles, .abcd/rules.json, internal/core/rules
- tests: the banned token on each root; the escape; the help and page grep

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 the sweep | scope 2 |
| 2 the lint | scope 1 |
| 3 the questions | scope 3 |
| 4 the glossary | scope 4 |
| 5 the test | scope 5 |
