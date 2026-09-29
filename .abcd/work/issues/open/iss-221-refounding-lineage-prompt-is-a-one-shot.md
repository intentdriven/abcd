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
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-28, lane drainAhoy): still real at ed4647299 (registerRepo asks the re-founding question through Prompter.Confirm, which a piped or --yes run answers with its No default, and no verb links lineage afterwards). Both remedies the record proposes are product choices: whether an install that cannot ask a person refuses the question, so a --yes run stops registering such a repository until someone answers, and whether a new verb links lineage after registration."
---

ahoy install's re-founding lineage prompt is a one-shot: under --yes/non-TTY it prints and silently takes the [y/N] default No, registers the root as new, and a second run reports already_up_to_date without re-asking — no verb links lineage after the fact, leaving two active index entries for the same project (old epoch 85573f…, new 488a0aa… after the attribution rewrite). Non-TTY identity prompts should refuse rather than default, and lineage should be linkable post-registration