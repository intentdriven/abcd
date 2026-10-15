---
id: dcn-2607130000000003
slug: kept-core-exists-bool-only-swallows
date: 2026-07-13
---
2026-07-13 (itd-85 M1): kept `core.exists` (bool-only, swallows errors) and
`ahoy.fileExists` (regular-file-only) as-is rather than folding all three
`exists` copies into `fsutil.Exists`. Chose partial consolidation over the
plan's full-consolidation because the other two hold different contracts;
merging them would smuggle a behaviour change into a behaviour-preserving
refactor. Only `lint.fileExists` (identical fail-closed contract) migrated.
