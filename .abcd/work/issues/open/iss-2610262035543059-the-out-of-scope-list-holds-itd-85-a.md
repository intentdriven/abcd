---
schema_version: 1
id: "iss-2610262035543059"
slug: "the-out-of-scope-list-holds-itd-85-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 608 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:67"
remedy: "none (filed automatically)"
finding_ref: cfk-62a7f061e192d168
---

The out-of-scope list holds itd-85, 'A Read-Only Repo-Conformance Audit', as later-phase draft work; the lint chapter ships that read-only conformance verb and cites itd-85 as its intent

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:67` — "A Read-Only Repo-Conformance Audit"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:184` — "[`itd-85`] (../../intents/drafts/itd-85-audit-verb.md)"

03-out-of-scope.md lists itd-85 among the drafts, the uncommitted bench of later-phase items. 16-lint.md describes the shipped, read-only `abcd lint` repo-conformance verb, with every target marked shipped, and names `itd-85` (still at its drafts path) as the intent it delivers. The capability cannot be both shipped and later-phase work that has not been committed to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 608.
