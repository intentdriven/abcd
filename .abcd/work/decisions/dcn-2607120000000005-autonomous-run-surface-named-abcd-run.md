---
id: dcn-2607120000000005
slug: autonomous-run-surface-named-abcd-run
date: 2026-07-12
---
2026-07-12 — autonomous-run surface named /abcd:run, taking itd-29's reserved
name as the host-delegated realization of its operator surface (not a parallel
/abcd:loop). Discovery: itd-29 (autonomous-run-resilience, planned) already owns
this surface over the ADR-27 run seam, already scopes out-of-band-merge/chain
reconciliation (host-owns-git MVP → future read-only `abcd run reconcile --json`)
and 429/quota (spc-35), and is deliberately deferred pending real evidence
(revisit trigger #5: two end-to-end autonomous runs). Sequence C→A: run the loop
as a plan+protocol under the harness loop now to dogfood + generate that evidence;
formalize commands/abcd/run.md + brief row + surface_coverage, reconciled into
itd-29, after 1-2 successful runs. Binary operator verbs (budget preflight, rewind,
ship, run reconcile) stay deferred in itd-29.
