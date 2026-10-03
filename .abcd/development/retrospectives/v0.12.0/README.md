---
release: v0.12.0
previous_release: v0.11.1
date: 2026-10-02
intents: [itd-24, itd-42, itd-60, itd-152, itd-2609081951381895, itd-2609201916056194, itd-2609212103568351, itd-2609212103572513, itd-2609212137129937, itd-2609221017023290, itd-2609221842494980]
audited: [itd-24, itd-42, itd-60, itd-152, itd-2609081951381895, itd-2609201916056194, itd-2609212103568351, itd-2609212103572513, itd-2609212137129937, itd-2609221017023290, itd-2609221842494980]
unaudited: []
audit_receipts: [rcp-e6bcb27c8050, rcp-7a2c62ab0228, rcp-121c23aff688, rcp-12ccf7627aac, rcp-ebf7d171b544]
---

# Retrospective for v0.12.0

What v0.12.0 shipped is in [its changelog section](../../../../CHANGELOG.md#0120---2026-09-30). Each intent's audit notes stay on the intent and are linked here, not repeated.

| Intent | Impact | Audit notes |
|---|---|---|
| [itd-24](../../intents/shipped/itd-24-reflect-command.md) Completed Releases Get A Retrospective | additive | [audit notes](../../intents/shipped/itd-24-reflect-command.md#audit-notes) |
| [itd-42](../../intents/shipped/itd-42-coherence-aware-grill.md) Grill Reads an Intent Against the Brief and Its Siblings, Not Just the Glossary | additive | [audit notes](../../intents/shipped/itd-42-coherence-aware-grill.md#audit-notes) |
| [itd-60](../../intents/shipped/itd-60-doc-fidelity-anti-drift.md) When A Surface Ships, The Brief Describes It — Or The Intent Does Not Reach Shipped | additive | [audit notes](../../intents/shipped/itd-60-doc-fidelity-anti-drift.md#audit-notes) |
| [itd-152](../../intents/shipped/itd-152-autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib.md) Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text. | additive | [audit notes](../../intents/shipped/itd-152-autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib.md#audit-notes) |
| [itd-2609081951381895](../../intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api-oracle-adapter-the-first.md) abcd ships an OpenAI-compatible API adapter, and a provider serves only the models it lists | additive | [audit notes](../../intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api-oracle-adapter-the-first.md#audit-notes) |
| [itd-2609201916056194](../../intents/shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a-command-line-model-run.md) A role runs through the command-line harness the operator chose, and every fall back to the host is recorded | additive | [audit notes](../../intents/shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a-command-line-model-run.md#audit-notes) |
| [itd-2609212103568351](../../intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site-s-status-page-show.md) The status board shows Now, Next and Later, computed from the record | additive | [audit notes](../../intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site-s-status-page-show.md#audit-notes) |
| [itd-2609212103572513](../../intents/shipped/itd-2609212103572513-an-intent-names-the-release-it-must-land-by-and-the-cut-says.md) An intent names the release it must land by, and the cut says whether it did | additive | [audit notes](../../intents/shipped/itd-2609212103572513-an-intent-names-the-release-it-must-land-by-and-the-cut-says.md#audit-notes) |
| [itd-2609212137129937](../../intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product-thinker-or-the-technical.md) abcd's text names the product thinker or the technical facilitator, never the maintainer | additive | [audit notes](../../intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product-thinker-or-the-technical.md#audit-notes) |
| [itd-2609221017023290](../../intents/shipped/itd-2609221017023290-abcd-keeps-every-external-credential-the-same-way-one.md) abcd keeps every external credential the same way | additive | [audit notes](../../intents/shipped/itd-2609221017023290-abcd-keeps-every-external-credential-the-same-way-one.md#audit-notes) |
| [itd-2609221842494980](../../intents/shipped/itd-2609221842494980-a-dependency-bump-lands-without-a-person-re-authoring-it-a.md) A dependency bump lands without a person re-authoring it, and no machine enters the contributor graph | additive | [audit notes](../../intents/shipped/itd-2609221842494980-a-dependency-bump-lands-without-a-person-re-authoring-it-a.md#audit-notes) |

## What went well

How we worked.

Reviews before planning: two adversarial reviews read every draft before the product thinker was asked, and they found flaws a sign-off would have missed. Plain, one-at-a-time questions, with the example inside the question. Research before ruling: primary sources were checked before a decision that rested on outside facts (the subscription question).

## What could improve

Drift from the promise: 9 of 11 features met their criteria only with concerns (7 diverged gaps, 1 missing). Checks running too late: 3 features shipped before their after-delivery audit, and a retired word reached a shipped press release and broke the site build. Sign-off and wording: a lane filed an intent as planned without the product thinker's sign-off, and the first interview questions were too technical.

Drift from the promise matters most; the other two follow.

## Lessons learned

- Review every draft twice before asking for a sign-off.
- Ask in plain words, with the example inside the question, even for technical decisions.
- Check the build against its promise before shipping, not afterwards.

## Decisions made

- Only the owner's personal settings may send work, files or roles to a paid or remote AI service or a command-line assistant; a project's own settings cannot.
- Every assistant abcd starts runs sealed from the project's own settings, instructions and add-ons (Claude Code with a paid API key only; opencode sealed and tested).
- A shipped feature that fails its after-delivery check stays shipped, flagged, with an issue filed to fix it and check again.
- Breaking changes wait for one breaking release (v0.12.0) that carries a migration note.

## Metrics

- Intents shipped: 11 (11 with audit notes, 0 without)
- Audit verdicts: MET 12 · MET_WITH_CONCERNS 17 · NOT_MET 0 · INCONCLUSIVE 0
- Gap audit: honoured 34 · diverged 20 · missing 3
- Released: v0.12.0 on 2026-10-01; the previous release, v0.11.1, on 2026-09-28
