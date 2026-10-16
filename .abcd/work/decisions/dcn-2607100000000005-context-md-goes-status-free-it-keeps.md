---
id: dcn-2607100000000005
slug: context-md-goes-status-free-it-keeps
date: 2026-07-10
---
2026-07-10 — CONTEXT.md goes status-free: it keeps orientation and the
live sharp-edges list only; hand-written phase/status claims are banned
(extending adr-5's no-status-in-design-docs rule to the work tier) and a
record-lint rule on .abcd/work/CONTEXT.md is the detector, armed before
the rewrite per fix-the-detector. The content rewrite rides with iss-35's
brief-vs-surface reconciliation. Rejected: deleting the file (loses the
only committed shared home for sharp edges); generating it (a committed
generated file is its own drift problem).
