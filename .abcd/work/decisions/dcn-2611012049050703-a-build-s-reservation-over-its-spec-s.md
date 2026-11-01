---
id: dcn-2611012049050703
slug: a-build-s-reservation-over-its-spec-s
date: 2026-11-01
---
2026-11-01 — A build's reservation over its spec's footprint follows a per-repository setting with two rules: as each step lands, narrow the reservation to the files the steps not yet landed still name (the default), or reserve only the steps in flight, widening as each lane opens and narrowing as each lands. The default never lets a later step's files go unreserved mid-run; the second blocks other copies least, at the cost of a build that can stall when its next step's files are taken. This answers the trust-rule hand-back of iss-2610230839002949.
