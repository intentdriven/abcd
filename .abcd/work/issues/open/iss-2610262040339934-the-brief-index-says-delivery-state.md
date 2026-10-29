---
schema_version: 1
id: "iss-2610262040339934"
slug: "the-brief-index-says-delivery-state"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 939 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:13"
remedy: "Replace 'roadmap dashboard' with the status board's Now / Next / Later block and the site's Status page (adr-2609212115255771); roadmap/README.md still carries the retired phase text."
finding_ref: cfk-f432529522008485
---

The brief index says delivery state lives in the roadmap dashboard; the roadmap glossary says the status block has replaced the roadmap and the word.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:13` — "lives in the roadmap dashboard, never in brief prose"
- End B: `.abcd/development/brief/glossary/core/roadmap.md:18` — "the rendered Now / Next / Later status block on the `abcd` board and the site's Status page replaces the roadmap document and the word."

The brief index sends readers to the roadmap dashboard for current delivery state. The roadmap entry is superseded: the Now / Next / Later status block on the abcd board and the site's Status page replaces both the roadmap document and the word.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 939.
