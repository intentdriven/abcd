---
id: dcn-2611161804516499
slug: the-command-facts-intent
date: 2026-11-16
---
2026-11-16 — The command-facts intent itd-2610301346085246 is brought to its full ac-1 before the v0.17.0 cut (a worked example for every visible command, and every closed-set flag declared through enumFlag so its allowed values reach the facts), rather than the criterion being narrowed to the spec's scope of examples for required-input verbs only; decided by the run for owner review

Grounds: The after-merge audit judged ac-1 NOT_MET against the intent's own words, which the product thinker confirmed on 2026-10-30, while the spec had narrowed examples to verbs with a required input and allowed values to flags whose help already called enumHelp. Narrowing a confirmed criterion is the product thinker's call, not the run's, and the audit's record is major, which the release cut refuses. The gap is mechanical: one example line per command, and moving hand-written closed-set help onto the existing helper, with tests holding both.

Reverse it by narrowing ac-1 in the intent to the spec's scope, re-running the audit with abcd intent audit, and reverting the examples and enumFlag commits that name iss-2611161802565435.
