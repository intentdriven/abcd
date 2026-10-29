---
schema_version: 1
id: "iss-2610262033223107"
slug: "05-intent-defers-the-honoured-diverged"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 397 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:553"
remedy: "In 05-intent.md (line 553 and the Role 1 text) say the honoured / diverged / missing gap audit ships in the audit verdict and its ingest (internal/core/intent/audit.go writes the three buckets into Audit Notes); keep only MG004 and the other prose and terminology outputs as deferred."
finding_ref: cfk-272747155f5b21ec
---

05-intent defers the honoured/diverged/missing gap buckets, but 09-reflect reads gap counts from every ingested review.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:553` — "(the `honoured` / `diverged` / `missing` buckets below) and other prose/terminology/PRD-fidelity outputs are **deferred** to a later spec"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:54` — "rollup an ingested review writes, and the honoured / diverged / missing gap"

05-intent records that the press-release prose review, meaning the honoured / diverged / missing buckets, is deferred to a later spec. It describes the audit ingest as writing only per-criterion verdicts and scope-condition dispositions. 09-reflect's seed takes, for each shipped intent, the acceptance rollup an ingested review writes plus the honoured / diverged / missing gap counts, and its metrics report the gap distribution. Either ingested reviews carry gap buckets or the seed has no gap counts to read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 397.
