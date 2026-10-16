---
id: dcn-2608200000000007
slug: the-spc-33-mint-mechanics-are-ruled-and
date: 2026-08-20
---
2026-08-20 — the spc-33 mint mechanics are ruled and the native mint ships
for captures: random-suffix width is 4 digits (16-digit ids, the top of the
range itd-114 priced — entropy alone carries the cross-branch same-second
case), and the same-instant tiebreak keeps the O_EXCL reservation but
REDRAWS a fresh id on clash rather than bumping (a bump is a miniature
max+1). The capture family now mints `iss-<yymmddHHMMSS><rrrr>` through the
family-generic recordid seam; itd/spc stay on the legacy refs-union
allocator until they adopt the seam as configuration. Capture's refs scan
and `mint_warning` retire with the maximum it warned about; the uniqueness
detectors stay armed as the scheme's fail-safe. Once this lands on main,
the per-task defensive minting protocols (fetch-and-verify twice, scanning
unmerged hunt ledgers) retire for captures, per adr-45.
