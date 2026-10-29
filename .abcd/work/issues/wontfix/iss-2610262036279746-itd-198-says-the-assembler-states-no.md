---
schema_version: 1
id: "iss-2610262036279746"
slug: "itd-198-says-the-assembler-states-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 649 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-198-an-assembly-reports-what-it-would-cost.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-198]
finding_ref: cfk-60d098de15cd4ed8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. assemble.go carries TargetTokens and OverTarget; the later intent names itd-198 and the ruling."
---

itd-198 says the assembler states no size figure of its own; the reading chapter has every assembly report against a standing size target

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-198-an-assembly-reports-what-it-would-cost.md:19` — "No budget is enforced and none is invented: the assembler cannot know what a given reader accepts, and a number it made up would be wrong for someone."
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:172` — "A standing size target, stated and never enforced."

itd-198 rules out any budget the assembler supplies, enforced or not, because it cannot know what a reader accepts and a number it made up would be wrong for someone. 23-reading has every assembly report a standing size target the tool states, naming both figures when the estimate runs over it. The two agree that nothing is enforced but disagree on whether the tool states its own size figure.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 649.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. assemble.go carries TargetTokens and OverTarget; the later intent names itd-198 and the ruling.
