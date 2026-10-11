---
schema_version: 1
id: "iss-2608270908348042"
slug: "five-delimiter-compare-variants-remain"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/frontmatter/frontmatter.go"
resolution: "The private delimiter walks route through frontmatter.Close, the new CloseAfter or IsDelimiter; the deliberate differences (memory's indented opener, the transcript store's own byte-exact format, the reading exclusion floor) are commented and allowlisted, and TestNoPrivateDelimiterCompare fails on a new private three-dash literal outside that allowlist. The mid-file ZWNBSP close in the re-verification note is refused by record-lint through the issue store's reader-parity leg, pinned by a case in TestRecordSchemaAgreesWithTheLedgerReader."
impact: internal
resolved_by:
  commit: "2b6e0bf8e"
---

five delimiter-compare variants remain beside the canonical frontmatter.IsDelimiter: gate-side TrimSpace compares in lint and glossary accept an indented delimiter the canonical rule refuses, intent and changelog carry tolerant local copies, memory keeps its own close predicate, and site tests a bare HasPrefix — one consolidation pass onto the canonical predicate closes the family
Re-verification note: a record whose block is closed only by a mid-file ZWNBSP delimiter is capture-refused but frontmatter.Fields-green, and no lint rule runs the strict ledger parser — record-lint passes what capture refuses until the consolidation lands.

## Grounds

- pursued: every reader of a record's frontmatter judges its delimiters by one rule, so a gate and its reader read the same block; a committed record whose block closes differently to a gate than to Fields, or a new private three-dash compare that the detector does not name, would show it wrong
