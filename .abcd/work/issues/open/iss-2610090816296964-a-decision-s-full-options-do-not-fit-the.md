---
schema_version: 1
id: "iss-2610090816296964"
slug: "a-decision-s-full-options-do-not-fit-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "an abcd question asked in another session on 2026-10-09"
origin: researcher-authored
production_mode: hand-written
found_at: "asking the person: the question tool and the guard's question check (internal/core/question)"
remedy: "Adopt a no-repetition rule now: the text before a question says only what its options cannot (what is being decided and where it stands), never restating an option, written into the GRILL asking rules and the abcd:question-drafter agent; then run a lab with the drafter on the quick tier (Haiku 4.5) rewriting until the row count fits, compared with the default tier on the same material for fit rate, lost meaning and cost (iss-2610080618506115)."
---

When abcd asks the person to choose, the question keeps running past the host's question box: a question over 24 rows at 80 columns was refused in a live session on the 0.13.2 plugin, and the agent cut its options down to a few words each to get it through. The product thinker, reading the refused draft, saw that its long text added nothing: a paragraph before the options described each option again, while the options' own labels and descriptions were already perfectly clear. The problem was repetition, not too little room, so the fix is a rule against it rather than a second channel for the overflow; a separate text file for the full options was considered and dropped on the product thinker's ruling of 2026-10-09.

The rule: the text before a question says only what its options cannot, which is what is being decided and where it stands now, and never restates an option's meaning, gain or cost. It belongs in the GRILL asking rules and in the abcd:question-drafter agent, which v0.13.3 ships to draft questions within the limit; the guard can count rows but cannot judge repetition, so the drafter is where it is enforced. v0.13.3 also stops refusing a question whose only fault is its length (iss-2610070637562567), which removes the refusal but not the repetition.

The lab that follows runs the drafter on the quick tier (Haiku 4.5), rewriting until the row count, computed exactly by abcd, fits the window, and compares its drafts with the default tier's on the same material: how often each fits, whether any meaning is lost, and the cost. Its result feeds the model-tier idea in iss-2610080618506115.
