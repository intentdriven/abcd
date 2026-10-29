---
schema_version: 1
id: "iss-2610262040433257"
slug: "itd-140-names-the-site-s-gate-abcd-site"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 951 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-140]
finding_ref: cfk-968d09a1051f9053
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035592770. Same sentence in itd-140 line 55, cited against 10-site.md instead of 16-lint.md."
duplicates: [iss-2610262035592770]
---

itd-140 names the site's gate `abcd site check`; the site chapter names it `abcd lint site`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:55` — "site check\` refusing a repo-agnostic claim without the fixture receipt"
- End B: `.abcd/development/brief/05-internals/10-site.md:72` — "**`abcd lint site`** runs seven independent gates over a rendered tree"

itd-140's armed-detector rung is `abcd site check`, refusing a repo-agnostic claim that has no fixture receipt. 10-site says the site's gates run through `abcd lint site`, seven independent gates over a rendered tree. That is two command names for the site gate, and the one the discipline names is not in the site chapter.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 951.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035592770. Same sentence in itd-140 line 55, cited against 10-site.md instead of 16-lint.md.
