---
id: spc-2609212139583822
slug: every-verb-s-help-opens-with-one-sentence-an-agent-can-act
intent: itd-2609212113220149
origin: researcher-authored
production_mode: hand-written
---
# every-verb-s-help-opens-with-one-sentence-an-agent-can-act

## Summary

The design record for itd-2609212113220149: one actionable sentence per verb from one source, held by a test.

## Scope

1. **The manifest** gains `sentence` per command; the CLI's `Short` and the pages' first line are generated from it (criterion 2).
2. **The form check**: a test parses each sentence into the three clauses by its declared separators (a colon after the doing clause; a semicolon before the refusing clause), enforces the cap (declared, 160 characters), and diffs the three renders (criteria 1, 3).
3. **The sweep**: fifty-three sentences rewritten to the form in one change, reviewed (criterion 1).
4. **The agent block** renders the same field (criterion 4).
5. **docs lint** includes the manifest's sentences as a lint root (criterion 5).

## Out of scope

- Long help; examples.

## Approach

The surface manifest is already the generated source of the pages and the snapshot; the sentence is one more generated field with a form test beside the snapshot test.

## Footprint

- packages: internal/core/surface, internal/surface/cli, commands/
- tests: the three-way diff; the clause parser on good and bad sentences; the docs-lint root

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 the form | scope 2, 3 |
| 2 identical from one source | scope 1 |
| 3 the test | scope 2 |
| 4 agent block | scope 4 |
| 5 docs lint | scope 5 |
