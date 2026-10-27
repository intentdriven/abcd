---
schema_version: 1
id: "iss-2610262033090287"
slug: "itd-146-puts-a-changelog-verb-in-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 367 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:902"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-9fdeb0d5ddf5e1e2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both statements were true when written and a later closed spec and the launch chapter say which holds; the shipped intent is a frozen record."
---

itd-146 puts a `changelog` verb in the agent help block; the launch chapter says the changelog verb is retired and refused as an unknown command

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:902` — "the retired changelog verb, the binary refuses it as an unknown command"
- End B: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:169` — "`changelog` sits in the agent block."

itd-146's decisions list `changelog` among the agent block's verbs and keep folding `changelog` under `launch` out of scope. The launch chapter says the preview renders the cut and there is no separate changelog verb. Its acceptance criteria have the binary refuse the retired changelog verb as an unknown command, naming the preview instead. A help layout that lists `changelog` and a binary that refuses it cannot both describe the shipped surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 367.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both statements were true when written and a later closed spec and the launch chapter say which holds; the shipped intent is a frozen record.
