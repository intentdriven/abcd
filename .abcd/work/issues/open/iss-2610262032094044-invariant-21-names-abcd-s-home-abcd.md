---
schema_version: 1
id: "iss-2610262032094044"
slug: "invariant-21-names-abcd-s-home-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 229 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:179"
remedy: "none (filed automatically)"
related_intents: [itd-2609212137128014]
finding_ref: cfk-5742241da292277e
---

Invariant 21 names abcd's home `~/.abcd.noindex`; itd-2609212137128014 records lab evidence at `~/.abcd/lab/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "the `.noindex` suffix the home carries, `~/.abcd.noindex`"
- End B: `.abcd/development/intents/shipped/itd-2609212137128014-abcd-lab-mechanises-the-lab-conventions.md:42` — "evidence at `~/.abcd/lab/`, knowledge through ceremony"

The brief puts abcd's home at `~/.abcd.noindex`. The lab intent sets the recorded convention that lab evidence lives at `~/.abcd/lab/`. The two place the same store in different folders.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 229.
