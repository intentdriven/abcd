---
id: spc-2610040847280931
slug: plain-terminal-interviews-remaining-work
intent: itd-2610030810370060
origin: researcher-authored
production_mode: hand-written
---
# plain-terminal-interviews-remaining-work

## Summary

This spec carries what [spc-2610030911534855](../closed/spc-2610030911534855-a-person-can-run-abcd-s-interviews-in-a.md) did not deliver for [itd-2610030810370060](../../intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md). Its four steps are built: the question type and the drawing, the answer loop, the fixed interviews with the answers record, and the AI-written interviews on the person's own route with the change guard around each turn. On the product thinker's rulings of 2026-10-04 (the decision log), the intent stays planned until three pieces land, and a fourth, found by the last security re-check, joins them:

- setup's answers file settles every question before setup writes anything, so criterion B3 holds when a file stops at a question decided during the run (iss-2610040025088395);
- the AI-written interviews in a Terminal take the person's own typed answer beside the role's drafts ("Add it to the remaining work");
- one real AI-written interview on the person's own route, run with the person, before the intent is marked done ("Add a real run to the remaining work");
- the change guard bounds a followed link's target, so a role cannot slow a turn for minutes before the guard refuses (iss-2610040847166532).

## Footprint

- packages: internal/core/ahoy, internal/surface/cli, internal/core/interview, internal/core/question, internal/surface/cli/ask, commands/, .abcd/development/brief/04-surfaces
- tests: TestSetupAnswersFileSettlesEveryQuestionBeforeAnyWrite, TestWrittenInterviewTakesATypedAnswer, TestAnswersFileCarriesATypedValue, TestAFollowedLinkIsBounded; the dated real-run receipt in the local tier

## Steps

1. Setup's answers file settles every question before any write
   - criteria: B3 in full: an answers file that does not answer a question decided during the run refuses before setup writes anything
   - packages: internal/core/ahoy, internal/surface/cli
   - tests: TestSetupAnswersFileSettlesEveryQuestionBeforeAnyWrite (a file stopping at a status-line element, an offer, lineage, house style and the artefact kind each refuses with the tree unchanged); the existing settle and walk-parity tests unchanged
   - waits on a question plan in core ahoy that names every question a run will ask, the ones decided during the run included; resolves iss-2610040025088395
2. Typed answers in the AI-written interviews
   - criteria: B5 (retrospective) in the person's own words
   - packages: internal/core/interview, internal/core/question, internal/surface/cli/ask, internal/surface/cli
   - tests: TestWrittenInterviewTakesATypedAnswer (the drawn loop offers the typed part beside the role's drafts and records the typed text, sanitised); TestAnswersFileCarriesATypedValue (replay by ordinal)
3. The change guard bounds a followed link
   - criteria: B6's guard stays prompt: a followed link's target is bounded by a small file and byte budget, refusing with the link named
   - packages: internal/core/interview
   - tests: TestAFollowedLinkIsBounded (a turn-directory receipts link to a large tree refuses quickly, naming the link); resolves iss-2610040847166532
   - landed: fix/guard-link-bound
4. One real run with the person, and the close
   - criteria: B5 and B6 on a real runner: one AI-written interview (a short planning interview on a draft intent) on the person's own route with the person; the guard does not stop on what the runner itself writes, or the fix lands in this step
   - tests: the dated receipt `.abcd/.work.local/logs/real-interview-<yyyy-mm-dd>.md`; then a docs-fidelity review recorded for HEAD and `abcd spec close spc-2610040847280931` (the intent declares impact additive) with a `Delivers: itd-2610030810370060` trailer
   - lands after steps 1 to 3
