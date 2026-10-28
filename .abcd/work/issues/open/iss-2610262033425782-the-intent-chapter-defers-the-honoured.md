---
schema_version: 1
id: "iss-2610262033425782"
slug: "the-intent-chapter-defers-the-honoured"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 431 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:553"
remedy: "none (filed automatically)"
finding_ref: cfk-3ee8aa951e299a99
---

The intent chapter defers the honoured/diverged/missing prose review to a later spec; the verification matrix says the shipped auditor prompt already emits it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:553` — "and other prose/terminology/PRD-fidelity outputs are **deferred** to a later spec."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:58` — "The shipped auditor prompt emits both halves: per-criterion verdicts over the acceptance criteria, each with a cited evidence pointer and a concerns verdict refused unless it names its concern, **and** the three-bucket prose audit."

05-intent says the press-release prose review, with its honoured / diverged / missing buckets, and the other prose outputs 'are deferred to a later spec'. The matrix's Intent audit (Role 1) row is not marked staged, so it describes what the binary carries today, and it says the shipped auditor prompt emits per-criterion verdicts 'and the three-bucket prose audit'. The bucket audit cannot be both deferred and shipped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 431.
