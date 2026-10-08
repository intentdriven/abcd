---
schema_version: 1
id: "iss-2610071538075997"
slug: "name-check-judge-warn-tier-hits-in-session-keep-a"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071222173136 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd lint docs (warn-tier banned tokens)"
remedy: "none (filed automatically)"
---

Name check: judge warn-tier hits in session, keep a local-model shadow log, and review it

## What this is

Three capabilities for abcd's name check, drafted as intents in a managed repository (pge) but belonging to the tool, not to that project. pge is only the first user. Its accepted decision (adr2610031742534388) settles how pge judges its own warn-tier hits; the features that decision needs should live in abcd so every repository can use them.

A regex cannot tell a protected game name from the same word used as ordinary English, so the warn tier needs a judge.

## The three capabilities

1. **In-session judging with a shadow log** (pge draft itd2610032153275513). The person runs the name check; for each warn-tier line the latest Opus in their session proposes "game reference" or "ordinary" with a one-line reason; the person approves or overrides. Approved game references are blocked; approved ordinary uses get the allow marker so the next run passes. Whenever the local model server is up, the local decision model's verdict and probability are logged beside the judge's proposal and the person's decision, and never change the outcome.
2. **Offline judging with a person in the loop** (pge draft itd2610032153311281, held until an offline or school use arises). Local decision model clears clear-ordinary lines, a local chat model with thinking off proposes the rest, and the person reviews every game-reference verdict.
3. **Shadow-log review** (pge draft itd2610032153313848). Reports how many judged lines the log holds, how often the local model would have allowed a confirmed game reference, a confidence bound on that miss rate, and whether the log now holds enough to revisit keeping the cloud judge in charge.

## Evidence behind it

From the lab run (lab-261002171217-c372ff6, held-out test of 280 flagged lines): the in-session frontier judge scored 97.8% with no missed game reference; the local decision model alone 90.0% with 10 misses; a live pipeline of local screen (zero-miss cut 0.05) then in-session judge scored 97.5% with no miss, the screen clearing 27% of lines. One test is not a track record, hence shadow mode rather than screen-first. Reasoning must be bounded on every judge request (open models spiralled at default reasoning).

## Decided so far

- **The shadow log is private to the machine** (owner's answer in pge's planning interview, 2026-10-07): kept in the local, uncommitted work tier, so flagged lines from a private repository never leave it; the accepted cost is that the evidence lives on one machine and is lost with the checkout.

## Still open

- What identifies a line so a log entry still matches after the line moves.
- Whether the allow marker records who decided, and when.
- For offline judging: should the person also review the local chat model's "ordinary" verdicts, since a missed game reference arrives as an ordinary verdict?
- For the review: what counts as "enough lines" (for example, the upper bound of the miss rate's interval below an owner-set figure), and whether it runs on demand only or is prompted after a set number of new entries.

## Dependencies noted in the decision

Serving the decision model through the local server depends on that server's decision-model support. abcd's provider adapter is non-streaming with a 120 s limit, so long delegated judging routed to a local provider is not yet reliable through abcd; the in-session judge avoids this.

Remedy the reporter proposes: Add an adjudication flow to the name check: an in-session judge proposes game-reference or ordinary per warn-tier line, the person approves, approved ordinary uses get the allow marker; a local decision model runs in shadow and is logged; a review reports the shadow miss rate.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071222173136, a enhancement against abcd v0.13.1, surface abcd lint docs (warn-tier banned tokens).

Evidence:

- rpt-2610071222173136 (the report, kept in the inbox)
- adr2610031742534388
- itd2610032153275513
- itd2610032153311281
- itd2610032153313848
- c3a5f13
- 490d14d
- lab-261002171217-c372ff6

Every record id the report names is its sender's own, not this repository's, so each is written as one word, family and number together, and cites nothing here.
