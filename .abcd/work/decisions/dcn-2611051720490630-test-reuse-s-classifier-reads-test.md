---
id: dcn-2611051720490630
slug: test-reuse-s-classifier-reads-test
date: 2026-11-05
---
2026-11-05 — Test reuse's classifier reads test source on a best-effort basis, so some spellings escape it, among them a program name held in a local variable (iss-2611051719413677). This is deferred as a design limit of a local speed-up, not a route to close before the cut: the hosted run reuses nothing and stays the authority on whether a change passes.
