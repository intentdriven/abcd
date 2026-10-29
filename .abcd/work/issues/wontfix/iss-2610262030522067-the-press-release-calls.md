---
schema_version: 1
id: "iss-2610262030522067"
slug: "the-press-release-calls"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 55 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-b079e142ce10f947
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Same press release sentence against another statement that the convention is not universal; one rewording of line 5 settles both."
duplicates: [iss-2610262030301422]
---

The press release calls bare-shows-status-and-next-actions universal; the intent chapter says it is common, not universal

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "Bare invocation of any command shows status + suggested next actions"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:344` — "Bare-command-as-help is a common abcd convention, not a universal one"

01-press-release states that bare invocation of any command shows status and suggested next actions as the universal abcd convention, while 05-intent says bare-as-help is a common convention, not a universal one, that several parents print help or usage with no board, and that suggested next actions are a design target, not a shipped guarantee.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 55.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Same press release sentence against another statement that the convention is not universal; one rewording of line 5 settles both.
