---
schema_version: 1
id: "iss-2610262036312405"
slug: "itd-104-says-every-ideate-run-outputs-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 654 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-104]
finding_ref: cfk-ce6174e190a607a8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/ideate.md requires exactly one of --private or --public, and a private verdict writes no decision because the decisions folder is committed; that split is a later shipped ruling the intent could not have named."
---

itd-104 says every ideate run outputs a verdict and a recorded decision; the ideate chapter writes no decision for a private verdict

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md:17` — "The output is a verdict and a recorded decision that keeps the rejected alternatives"
- End B: `.abcd/development/brief/04-surfaces/18-ideate.md:88` — "no decision points at it, because the decisions folder is committed"

itd-104 states the output of the admission interview as a verdict plus a recorded decision keeping the rejected alternatives. 18-ideate writes a private verdict only under the home store, with no decision pointing at it because the decisions folder is committed. For a private run the recorded decision itd-104 promises does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 654.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/ideate.md requires exactly one of --private or --public, and a private verdict writes no decision because the decisions folder is committed; that split is a later shipped ruling the intent could not have named.
