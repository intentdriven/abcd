---
schema_version: 1
id: "iss-2610140652230444"
slug: "every-review-runs-as-a-two-class-pair"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "cut 2 autonomous run, 2026-10-14 (paired Opus/Fable/Sonnet security reviews)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md"
refines: [itd-2609170822093401]
remedy: "Make the review set a setting, in vendor-neutral model classes: by default one reviewer model; optionally two in parallel on the same frozen copy and brief, with a third model as judge or decider that is consulted only when their verdicts disagree; each vendor's models map onto the classes in configuration, never in the rule; and record every review's findings per reviewer (which found what, where they overlap and where they differ) so the default is chosen from measured catch rates, not assumed."
related_intents: [itd-2610161733316518]
---
Every review should run as a pair of models from two classes, with the top class only an occasional advisor. The classes are vendor-neutral: top, middle, the class below it, and smallest (the scale itd-2610100704270254 uses). For one vendor today, Anthropic, they are Fable, Opus, Sonnet and Haiku. The rule is stated in classes so it survives model churn and holds for any vendor whose models map onto them.

The rule, as the product thinker ruled it on 2026-10-14:

- Every review, security or code, runs as a pair: a middle-class reviewer and a reviewer from the class below it, in parallel, on the same frozen copy with the same brief. The stricter verdict decides, and every finding is kept.
- Code is written by the middle class.
- The top class is a consult-only advisor, consulted in exactly four cases, which the product thinker confirmed the same day ("yes, those are exactly the cases where we need fable"):
  1. a third read of release-blocking security changes: the shell guard, installing and updating the binary, secrets, and what is published;
  2. before a design-heavy plan locks;
  3. when the executor fails twice with the same failure fingerprint;
  4. when the pair's verdicts disagree.

Evidence from the cut 2 run of 2026-10-14. Three security reviews were each run on a model from each of the three upper classes, with the same frozen input and the same brief, and each model caught a finding the other two missed:

- The middle class alone found a guard regression that blocked the cut: a known-path write followed by a source in an unknown directory, allowed after a fix.
- The class below found a major masking gap: most run-state string fields were unmasked, confirmed by a probe in `implement record --json`. It was also the fastest and cheapest on every input.
- The top class found that a relative write followed by a run of the same name in an unknown directory only warned where it should block. It was the slowest and most expensive on every input, and none of its unique findings blocked the cut.

Five later gate reviews used the pair alone. The middle class caught every blocker; the class below matched it twice and missed twice, both on shell semantics. Published evidence checked the same day does not show the top-class model stronger than the middle-class one at review, and no source compares them on security review.

Today the rule lives only in one session's local memory, so it reaches no other agent or contributor.

## Addition 2026-10-16 (product thinker): the review set is a setting

The product thinker asked for the number and kind of reviewers to be configurable rather than fixed as a pair. By default one reviewer model runs. A person can instead set two models to run in parallel, with a third model as judge or decider that is consulted only when the two disagree. The remedy above now says so. The pair has been the rule since 2026-10-14, so this addition also records how the pair has done.

## Evidence 2026-10-16: what each class found, per review

No reviews were logged systematically before this addition. These rows are taken from the reports of one coordinating session's reviews on 2026-10-16, each pair run on the same frozen copy with the same brief. "Middle" is the middle class (Opus 5.5 here), "below" the class beneath it (Sonnet 5.5).

| Change reviewed | Middle | Below | Overlap | Only one found it |
|---|---|---|---|---|
| Load gate queue order | ship | ship with fixes | none blocking | below: a just-started waiter's zero age contradicts the tie rule (a test was added); middle: a doc simplification and a stale test comment |
| Shell `-c` delete refusal, first round (run by another session) | ship | fix first | both: a false refusal of a literal-path delete | middle: four forms the refusal does not read (each now its own issue) and a replay of 2,589 commands; below: 34 test assertions no longer reaching the live verdict |
| Shell `-c` delete refusal, re-check | ship | ship | none | middle: an untested literal-`cd` case (a test was added); below: a file-fed delete not read (an issue was filed) |
| Default-branch helper | ship | ship | none | middle: a test that never varies the remote, and a third copy of the lookup (both fixed); below: two untested orderings |
| Messaging step 3, fix round 3 (this session's pair) | fix first | ship | none | middle: the intake cap dropped whole branches' announcements for good. Below reviewed the same lines and judged them the documented design |
| Messaging step 3, fix round 3 (a second session's pair, same copy) | fix first | fix first | both: the same intake-cap defect | three minors, attributed to the pair jointly |
| Headless approvals, security round | fix first | ship | none | middle: a working directory kept between shell calls lets two allowed commands write inside `.git`, plus missing writers and unprotected trust files; below: two older false refusals (`cd sub && rm x.o`, flags after a script name) |

What the rows suggest, and do not yet show:
- The middle class found every blocking defect: three of the seven reviews. The class below found one of the three: the same intake-cap defect in the second session's run, after missing it in the first.
- The class below's findings were real, mostly false refusals, edge cases and test gaps, and in four of the seven reviews it found something the middle class did not.
- **The same class can differ from run to run.** On the same frozen copy, one run of the class below missed the intake-cap defect and another found it. Some of what looks like a difference between models is variance between runs, so one review is a sample, not a measurement.
- The third model as decider would have been consulted in four of the seven reviews (the verdicts differed in the load-gate queue order, the first shell refusal round, step 3's first pair and headless approvals). In each of the four, the stricter verdict, which the current rule already takes, held up: its findings were real and were fixed.

To make this a measurement, each review should write one row: the change and frozen copy, each reviewer's class, model and verdict, and each finding with its consequence, plus which other reviewer also found it.
