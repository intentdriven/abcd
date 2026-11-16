---
id: dcn-2611161928510891
slug: the-command-page-facts-lint-keeps-the
date: 2026-11-16
---
2026-11-16 — The command-page facts lint keeps the spec's shape, a pure function in internal/core/lint run by a test in internal/surface/cli over the committed pages, and takes the cross-package test seam's name, CommandPageFactsForTest, to satisfy the exported-reach audit, rather than gaining an abcd lint target in this cycle; decided by the run for owner review

Grounds: The reach audit refuses an exported core function no production code outside its package reaches, and names a cross-package test seam ...ForTest; the spec placed the lint in a test because internal/core/lint cannot import the command tree. An abcd lint target would be a new public surface, needing its own facts, page and brief text, which is a change of its own after the cut; the test already fails the preflight on any finding.

Reverse it by wiring a front door (an abcd lint target or the command-facts sync check) that calls the function, and renaming it back to CommandPageFacts.
