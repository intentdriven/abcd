---
schema_version: 1
id: "iss-2610262042028384"
slug: "the-site-chapter-says-no-repo-agnostic"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1042 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/10-site.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-2609061543533170]
finding_ref: cfk-402cfd248ecdc296
---

The site chapter says no repo-agnostic explorer claim is made until a second instance proves it; shipped itd-2609061543533170 claims the pages render for every managed repository.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/10-site.md:109` — "the gate adr-47 decision 6 puts in front of any repo-agnostic claim, and the reason none is made here"
- End B: `.abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:43` — "landing, explorer, record pages, graph, timeline, glossary and status render for every managed repository from its own text"

10-site holds that genericity is demonstrated on a sparse second instance before it is claimed. It names itd-139's second-instance demonstration as the gate adr-47 sets in front of any repo-agnostic claim, and 'the reason none is made here'. The shipped site-setup intent makes that claim outright: the landing, explorer, record pages, graph, timeline, glossary and status render for every managed repository from its own text.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1042.
