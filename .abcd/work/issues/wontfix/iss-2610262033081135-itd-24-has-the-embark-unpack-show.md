---
schema_version: 1
id: "iss-2610262033081135"
slug: "itd-24-has-the-embark-unpack-show"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 364 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:61"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-c498e8e2d3efa487
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's press-release wording predates the build; its audit notes already say which verb surfaces the retrospectives."
---

itd-24 has the embark unpack show earlier retrospectives in the press-release interview; the embark chapter gives that job to the separate lessons sub-verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:61` — "for the press-release interview to ask which apply"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:55` — "`/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview"

itd-24 says `/abcd:embark from <path>` (the unpack) surfaces earlier retrospectives during the press-release interview, and its press release says embark surfaced them in the lifeboat unpack. In the embark chapter, the unpack writes retrospectives verbatim and reports blanks, writes, counts and the marker action. Surfacing them belongs to `lessons`, a separate read-only sub-verb that ranks the retrospectives' lessons for the press-release interview. The two documents disagree on which verb a person runs to see what the previous voyage learned.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 364.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's press-release wording predates the build; its audit notes already say which verb surfaces the retrospectives.
