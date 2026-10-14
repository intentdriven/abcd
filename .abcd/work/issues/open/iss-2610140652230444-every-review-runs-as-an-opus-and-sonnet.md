---
schema_version: 1
id: "iss-2610140652230444"
slug: "every-review-runs-as-an-opus-and-sonnet"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "cut 2 autonomous run, 2026-10-14 (paired Opus/Fable/Sonnet security reviews)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md"
refines: [itd-2609170822093401]
remedy: "Give the tier table's reviewer roles a paired form: each review (security and ruthless) runs on two tiers in parallel, Opus and Sonnet, on the same frozen copy and the same brief, and the loop merges both verdicts (the stricter one decides, every finding is kept); a top-tier advisor (Fable today) is consulted only in named cases, as the advisor role planned for this intent; measure the pair's catch rate per finding before the rule hardens."
---

Every review should run as a pair, Opus and Sonnet together, with Fable only an occasional advisor. The product thinker ruled this on 2026-10-14, after three security reviews in the cut 2 run were each run on all three models with the same frozen input and the same brief, and each model caught a finding the other two missed: Opus alone found a guard regression that blocked the cut (a known-path write followed by a source in an unknown directory, allowed after a fix); Sonnet alone found a major masking gap (most run-state string fields unmasked, probe-confirmed in implement record --json), and was the fastest and cheapest on every input; Fable alone found that a relative write then a same-name run in an unknown directory only warned where it should block, and was the slowest on every input. Published evidence (Anthropic's own benchmarks and model guidance, checked the same day) does not show Fable 5.1 stronger than Opus 5.5 at review; no source compares the models on security review. The proposed occasional cases for the advisor, pending the product thinker's confirmation: a third read of release-blocking security changes; before a design-heavy plan locks; when the executor fails twice with the same failure fingerprint; when the pair's verdicts disagree. Today the pairing lives only in one session's local memory, so it reaches no other agent or contributor.
