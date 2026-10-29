---
schema_version: 1
id: "iss-2610262036193171"
slug: "the-site-chapter-s-setup-takes-any"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 639 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-2610032150577708]
finding_ref: cfk-c3fcc2a6fe2b7e0a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The planned site intent's decisions say verbatim that abcd must provide the record page for any abcd-managed repo, which post-dates and overrides decision 13, and setup is shipped for managed repositories."
---

The site chapter's setup takes any managed repository's site live and gives every repository the same pages; itd-2610032150577708 rules that the record site is promised for abcd's own record only.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/22-site.md:48` — "Setting up takes a repository abcd manages from the checkout to a live address"
- End B: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "The record site and the dashboard are promised for abcd's own record only"

22-site says setup takes a repository abcd manages from the checkout to a live address in three stages, and that every repository gets the same page set. Decision 13 of itd-2610032150577708 gives up itd-140's promise that the record site works for any project, and says the record site and the dashboard are promised for abcd's own record only. The shipped surface promises what the ruling withdraws.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 639.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The planned site intent's decisions say verbatim that abcd must provide the record page for any abcd-managed repo, which post-dates and overrides decision 13, and setup is shipped for managed repositories.
