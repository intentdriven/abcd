---
id: dcn-2608300000000021
slug: itd-179-delta-close-mdrecord
date: 2026-08-30
---
2026-08-30 — itd-179 delta close: `mdrecord.findBacktickRun`'s superlinear shape stays, on a measurement rather than a shrug. A line of 120 distinct-length backtick runs costs ~200us and an ordinary record line ~76ns; precomputing the runs into a slice takes the bad line to ~7us and the ordinary line to ~115ns with one allocation per line, and stepping between runs with `strings.IndexByte` takes the bad line to ~243us. Both trade the case that always happens for a case no record body has. Rejected: landing the precomputed-run version, which reads as an optimisation and is a regression on this repo's workload. The two benchmarks and the numbers live in the package so the trade is not re-litigated.
