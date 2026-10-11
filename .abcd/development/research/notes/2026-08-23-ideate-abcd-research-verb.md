# Ideate verdict — abcd-research-verb

**Verdict: killed.** Recorded on 2026-08-23 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

adding a new /abcd:research verb to the abcd corpus

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation.

| Claim | Primary source | Finding |
|---|---|---|
| Survey-shaped SOTA research recurs in this repo as an established genre (14 *sota* notes since 2026-07-06) | .abcd/development/research/notes/ directory listing | verified |
| The /abcd:ideate schema cannot hold a survey: one idea in, ordered legs, mandatory survives/killed/reframed verdict | commands/ideate.md | verified |
| ideate's own validator rung was admitted after three manual protocol runs on 2026-07-14, so a small run-count is no bar to a validator rung | .abcd/development/intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes.md | verified |
| The survey-note schema a validator would enforce is unsettled: the two 2026-08-22 sota notes each declare a different evidence-tier ladder | .abcd/development/research/notes/2026-08-22-context-window-management-sota.md and 2026-08-22-local-models-mlx-sota.md (headers) | verified |
| No existing surface validates a survey note: consult/ingest cover the sources corpus and provenance; docs-lint/record-lint check currency and structure, not review presence | commands/consult.md, commands/ingest.md | verified |
| Review outcomes are now recorded in-tree, giving a validator something checkable | Review record sections of the three 2026-08-22 research notes | verified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| itd-104 | covered | Partial coverage only: its leg 1 covers claim-scoped admission research; survey conduct and note validation are outside its schema. Adjacent precedent for the validator anatomy (host-run legs, native record validator). The script-first-mvp principle (no per-entry id) bears on timing: the documented protocol rung precedes automation. The leg-3 evaluator additionally surfaced .abcd/development/research/notes/2026-08-22-sota-research-protocol.md (id-less, so uncitable here), whose standing verdict on this exact idea is 'not yet' with an evidence-gated revisit condition that has not fired. |
| adr-25 | contradicted | Contradicts a natively-executing research verb only: research execution is host-delegated and already shipped as the plugin-bundled sota-researcher agent; a validator-only rung is unaffected. |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.

- **fatal** — Already litigated: the repo recorded 'not yet' on this exact idea one day earlier with an evidence-gated revisit condition (hand-run drift), and no leg finding shows the condition fired - the newest notes exhibit compliance (inline review records) and the tier-ladder divergence is explicitly declared calibration data, so running the promotion path without its trigger is the re-litigation the record forbids
- **fatal** — No one has the problem: hand-run protocol plus lint gates plus in-tree review records already make a skipped review detectable; frequency without a settled contract is not the trigger (itd-104 minted against a settled contract)
- **survived** — Verb is the wrong shape: execution is host-delegated and shipped, corpus side is covered by consult/ingest, leaving only a validator
- **fatal** — Unsettled schema kills minting now: both notes declare per-note ladders as deliberate calibration; a validator canonising a ladder would contradict the corpus's explicit position, and even a weak structural contract (step 6) is one day old
- **partial** — Permanent surface cost for a low-frequency genre (22nd verb; CLI+plugin wiring, tests, docs, lint forever) while the claimed feeder payoff is an undemonstrated expectation
- **survived** — Scope creep toward native execution colliding with adr-25

## Rejected alternatives

- **Fold research into ideate as a pre-leg or sub-mode** — Schema mismatch: a survey has no single idea, no verdict, and no claim table until the survey is done; forcing it through produces pseudo-verdicts
- **A natively-executing research verb (fetching and synthesising in the binary)** — Contradicts adr-25: research execution is host-delegated and already shipped as the bundled sota-researcher agent
- **A validator-only rung (abcd research record) minted now** — script-first-mvp contract uncertainty: the note schema is declared calibration-in-progress and the in-tree review-record contract is one day old; the standing record already names this the candidate and defines the revisit trigger, so minting now re-decides a decided question

## What follows

The idea is closed. Before proposing it again, read this record: the
findings above are why it did not survive, and re-proposing it costs
nothing only if something above has changed.
