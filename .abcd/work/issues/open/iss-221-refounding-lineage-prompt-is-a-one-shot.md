---
schema_version: 1
id: "iss-221"
slug: "refounding-lineage-prompt-is-a-one-shot"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "manual-capture"
found_at: "internal/core/ahoy"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-28, lane drainAhoy; rulings-owed AV): still real at ed4647299 (registerRepo asks the re-founding question through Prompter.Confirm, which a piped or --yes run answers with its No default, and no verb links lineage afterwards). Both remedies the record proposes are product choices: whether an install that cannot ask a person refuses the question, so a --yes run stops registering such a repository until someone answers, and whether a new verb links lineage after registration."
remedy: "Waits on ruling AV (re-founded repository with nobody to answer): whichever answer, add an install flag that answers the lineage question up front (link to a named old root, or register as new) so no run requires the prompt; if refused: under --yes or a non-TTY stdin, internal/core/ahoy registerRepo leaves a re-founding candidate unregistered and names that flag; if registered as new: add a verb that links lineage after registration; proven by an ahoy test with the RefusingPrompter for each branch."
---

ahoy install's re-founding lineage prompt is a one-shot: under --yes/non-TTY it prints and silently takes the [y/N] default No, registers the root as new, and a second run reports already_up_to_date without re-asking — no verb links lineage after the fact, leaving two active index entries for the same project (old epoch 85573f…, new 488a0aa… after the attribution rewrite). Non-TTY identity prompts should refuse rather than default, and lineage should be linkable post-registration

## Remedy grounds (2026-09-29)

- The refusal-plus-flag shape follows the Command Line Interface Guidelines: only prompt when stdin is a TTY, and never require a prompt, always offering a flag for the same input (https://clig.dev/, checked 2026-09-29).
- The flag is common to both answers of ruling AV, so the remedy names it without choosing between refusing and linking afterwards.
- Rejected: keeping the silent No default, which is the behaviour the record reports.
