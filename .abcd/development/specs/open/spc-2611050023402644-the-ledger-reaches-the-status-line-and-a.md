---
id: spc-2611050023402644
slug: the-ledger-reaches-the-status-line-and-a
intent: itd-2611031342011261
origin: researcher-authored
production_mode: hand-written
---
# the-ledger-reaches-the-status-line-and-a

## Summary

_Draft: describe what spc-2611050023402644 delivers for itd-2611031342011261 — scope, approach, and how it satisfies the intent's Acceptance Criteria. This spec is the design record the fidelity review audits against._

## Footprint

- packages:
- tests:

## Steps

1. AFTER: the status line's row, its cached read and its explanation, with the token basis as a setting
   - packages: internal/core/statusline, internal/surface/cli
   - tests: the row from the ledger alone, within the status command's deadline, cached for the configured interval
   - needs: none
2. AFTER: an unattended run's summary carries the ledger's totals
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: a drain's and a driven build's summary naming their totals and the verb
   - needs: none
