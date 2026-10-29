---
schema_version: 1
id: "iss-2610262038389473"
slug: "itd-1-says-audit-notes-fill-only-from-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 808 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-8a2f987a7be8f142
refines: [iss-2608290820473197]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against 01-agents.md instead of 34-build.md."
duplicates: [iss-2610262037138873]
---

itd-1 says Audit Notes fill only from a hand-run audit; 01-agents says a build run audits automatically in the validate stage

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "Until that lands, a shipped intent's `## Audit Notes` is populated only when `/abcd:intent audit` is run by hand."
- End B: `.abcd/development/brief/05-internals/01-agents.md:156` — "A `build` run audits in the implement loop's validate stage, on the lane whose landing closes the intent's last open spec and before that lane lands"

itd-1 defers automatic invocation on the planned-to-shipped move and says that until it lands a shipped intent's Audit Notes are populated only when /abcd:intent audit is run by hand. 01-agents says a build run audits in the implement loop's validate stage before the lane that closes the intent's last spec lands. The audit cannot be both hand-run only and run by the build loop.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 808.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037138873. Same stale sentence in itd-1 line 45, cited against 01-agents.md instead of 34-build.md.
