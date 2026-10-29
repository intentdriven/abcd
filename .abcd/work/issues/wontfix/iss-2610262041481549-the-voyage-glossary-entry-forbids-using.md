---
schema_version: 1
id: "iss-2610262041481549"
slug: "the-voyage-glossary-entry-forbids-using"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1026 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md:59"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-75e308e01e9a27d9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and adr-35 names the later ruling; the glossary's open-question note already tracks the unreplaced project term."
---

The voyage glossary entry forbids using voyage for a project; itd-36 uses voyage to mean the project a researcher runs under abcd.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/voyage.md:59` — "Do not use it for a "project" (too generic)"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:23` — "Researchers and engineers running a `voyage` under abcd can drop external sources"

The voyage entry defines a voyage as the operations namespace under ~/.abcd.noindex/voyage/ that records disembark and embark runs. It says not to use the word for a 'project', and adr-35 retired the project-lifecycle sense. itd-36 uses it in that retired sense: a per-voyage substrate, and researchers 'running a voyage under abcd' who drop sources 'into the voyage'.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1026.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and adr-35 names the later ruling; the glossary's open-question note already tracks the unreplaced project term.
