---
id: dcn-2607130000000007
slug: docs-currency-emits-every-finding-at
date: 2026-07-13
---
2026-07-13 (itd-85 M3): docs-currency emits every finding at warn, downgrading
docs-lint blockers, because audit is an advisory conformance surface and the
authoritative docs gate is `abcd docs lint` (still exits 2 on a blocker).
Re-raising a docs blocker as an audit error would double-gate the same check.
