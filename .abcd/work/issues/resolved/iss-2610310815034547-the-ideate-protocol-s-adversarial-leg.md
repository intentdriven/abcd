---
schema_version: 1
id: "iss-2610310815034547"
slug: "the-ideate-protocol-s-adversarial-leg"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Add a current-state choice to ideate that defaults to ignore. /abcd:ideate takes it from the product thinker (e.g. --current-state ignore|consider), and the verdict payload carries it as a required field that abcd ideate record validates and renders. Under ignore, which is the default, commands/ideate.md has every reviewer brief treat the idea as an ambition for the product the repository builds. A gap between the idea and what that product ships today is out of scope as a kill attempt. Reviewers attack the ambition itself: whether the need is real, whether it holds against converging competitors, whether it is coherent with the rest of the product's ambition, and the wording. They list the gaps as what the idea implies building. Under consider, current delivery is in scope, as today. Word the rule in terms of the repository's own product and record (its brief, identity block and intents), never abcd by name, so it holds in any abcd-managed repository. Prove it with a record test that a payload without the field is refused and that each value renders, and a surface test that the page carries both modes and the default. Grounds: internal, the product thinker's 2026-10-31 rulings."
resolution: "ideate's adversarial leg now runs under a current-state mode the person chooses with --current-state ignore|consider, ignore by default: under ignore the reviewers judge the idea as an ambition for the product the repository builds and list a delivery gap as what it implies building, never as a kill. The verdict payload carries current_state as a required field that ideate record validates, and the record and the result state the mode."
impact: additive
---

The ideate protocol's adversarial leg gives reviewers no way to tell an ambition from a present-tense claim, so an idea about where a repository's product should go is killed because the product does not deliver it yet. On a 2026-10-31 run on a positioning idea (in abcd's own repository, where abcd is the product), both reviewers marked it fatal on current delivery gaps. The product thinker ruled that not yet built is never a reason to kill an ambition: the gap is the roadmap the exercise exists to find.

They added two requirements. First, the product thinker chooses whether the review considers the product's current state, and by default it does not. Second, the rule holds for the product any abcd-managed repository builds, so a product thinker can ideate around their own product and not only around abcd; abcd's own repository is the proving ground, not the scope.

## Grounds

- pursued: reviewers told to ignore current delivery stop killing ambition ideas on not-yet-built gaps; a later ignore-mode verdict whose fatal attempt rests on a delivery gap would show it wrong
