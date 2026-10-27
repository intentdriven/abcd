---
schema_version: 1
id: "iss-2610262035181369"
slug: "itd-43-rules-its-reviews-rename-moot"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 562 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-43]
finding_ref: cfk-c3c17a64efa1df26
---

itd-43 rules its reviews rename moot because no reviews subsystem classifies reviews by kind; the status-board chapter describes a reviews reader whose rows carry a kind and a spec

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:41` — "the Go tree has no reviews subsystem that classifies reviews by kind, so none of these surfaces exists to rename."
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:311` — "each row `folder`, `kind` (`review` or `receipt`), `scope`, `spec` when the name carries one"

itd-43 says the Go tree has no reviews subsystem that classifies reviews by kind, so there is nothing to rename. The status-board chapter describes a reviews reader whose rows each carry a kind (review or receipt) and a spec read from the folder name. The premise behind the moot ruling no longer matches the surface the brief documents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 562.
