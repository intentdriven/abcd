---
schema_version: 1
id: "iss-2610262032071046"
slug: "the-naming-register-says-current"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 225 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:161"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103568351]
finding_ref: cfk-c0fa7074511359d8
---

The naming register says current delivery state lives in 'the roadmap dashboard'; itd-2609212103568351 retired the word roadmap and calls the block status.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:161` — "Current delivery state lives in the roadmap dashboard, never here."
- End B: `.abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:58` — "The word roadmap is retired; this block is called status."

The register points a reader to a roadmap dashboard for current delivery state. The shipped status-board intent rules that the word roadmap is retired, that it appears on neither surface, and that the block is called status. The brief sends readers to a surface under a name the record retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 225.
