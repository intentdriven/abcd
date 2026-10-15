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
remedy: "Give the tier table's reviewer roles a paired form in vendor-neutral model classes (top, middle, the class below it, smallest, as itd-2610100704270254 names them): every review (security and code) runs as a pair, one reviewer from the middle class and one from the class below it, in parallel on the same frozen copy with the same brief; the stricter verdict decides and every finding is kept; code is written by the middle class; the top class is a consult-only advisor in exactly four cases: a third read of release-blocking security changes (shell guard, binary install and update, secrets, what is published), before a design-heavy plan locks, when the executor fails twice with the same failure fingerprint, and when the pair's verdicts disagree; each vendor's models map onto the classes in configuration, never in the rule; measure each pair's catch rate per finding before the rule hardens."
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
