---
schema_version: 1
id: "iss-2610262043369188"
slug: "itd-140-makes-the-landing-page-opt-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1149 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:29"
remedy: "In itd-140 rule 2 (line 29) remove the landing page from the opt-in list: it is the site's root and cannot be switched off; keep the cast imagery, install tab row, install.sh and deploy workflow as the opt-in specific side."
related_intents: [itd-140, itd-2609061543533170]
finding_ref: cfk-d385a76a93971c41
---

itd-140 makes the landing page opt-in and never a default; itd-2609061543533170 renders it for every managed repository and forbids switching it off

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:29` — "the deploy workflow: opt-in, declared in the repo's `.abcd/site.json`, never a default"
- End B: `.abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:68` — "the landing page and the record pages cannot be switched off beneath the explorer"

itd-140 places the landing page on the specific side of the site boundary: opt-in, declared in the repository's site.json, never a default. The shipped site intent renders the same page set for every managed repository, opt-out per page, and rules that the landing page cannot be switched off beneath the explorer. A landing page cannot be both opt-in and impossible to switch off.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1149.
