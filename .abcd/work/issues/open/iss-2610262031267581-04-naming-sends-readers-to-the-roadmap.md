---
schema_version: 1
id: "iss-2610262031267581"
slug: "04-naming-sends-readers-to-the-roadmap"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 130 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:161"
remedy: "none (filed automatically)"
finding_ref: cfk-acaab853bbfd4a66
---

04-naming sends readers to the roadmap dashboard for delivery state; the roadmap glossary entry says the Now / Next / Later status block replaces the roadmap document and the word.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:161` — "Current delivery state lives in the roadmap dashboard, never here."
- End B: `.abcd/development/brief/glossary/core/roadmap.md:18` — "replaces the roadmap document and the word"

04-naming's note on the reserved-vocabulary table says 'Current delivery state lives in the roadmap dashboard, never here.' The glossary's roadmap entry is superseded: the rendered Now / Next / Later block on the abcd board and the site's Status page 'replaces the roadmap document and the word'. The naming register points at a retired artefact by a retired name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 130.
