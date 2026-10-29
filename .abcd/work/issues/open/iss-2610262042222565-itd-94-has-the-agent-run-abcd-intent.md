---
schema_version: 1
id: "iss-2610262042222565"
slug: "itd-94-has-the-agent-run-abcd-intent"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1065 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:23"
remedy: "In plan.md Examples (line 63) reword to: 'intent ready exits 1, so the intent cannot be implemented; offer the planning interview, and run plan only after the human explicitly confirms the criteria (the human types it in a Terminal, or the host runs it on that confirmation)'."
related_intents: [itd-94]
finding_ref: cfk-45277ccff902257e
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-94 has the agent run `abcd intent plan` as the person's sign-off; the plan entry has the human run it and says it is never inferred from consent.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:23` — "Only then does the agent run `abcd intent plan` — your sign-off act"
- End B: `.abcd/development/brief/glossary/core/plan.md:63` — "offer the planning interview and let the human run `plan`"

itd-94 says that once the person has confirmed the interview, the agent runs `abcd intent plan` as their sign-off act. The plan entry says the invocation itself is the product thinker's sign-off, is never run unattended and is never inferred from consent. Its worked example is to offer the planning interview and let the human run `plan`. The two documents disagree on who carries out the sign-off.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1065.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
