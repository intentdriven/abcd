---
schema_version: 1
id: "iss-2609210748032488"
slug: "the-cold-reading-window-eval-fails-on-every-branch-that-grow"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "pilot run 2026-09-20, orchestrator"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_window_test.go"
---

The cold-reading window eval fails on every branch that grows the intent, capture or lint surfaces, and the recalibration it names is a hand-run recipe no verb performs. TestEveryCommittedEntryFitsItsDeclaredWindow (evals/coldreading_window_test.go, in the smoke lane, so in make preflight and the pre-push hook) re-measures each committed preset entry against the tokens_est it declares, and the widening and detection objects list internal/core/intent, internal/core/lint, commands/intent.md and commands/capture.md by path, so any change on the intent lifecycle moves the measurement; the declared margin is one per cent, about nine thousand tokens on a nine-hundred-thousand-token object, which one lane consumes. Measured in the pilot run of 2026-09-20: main at the pilot base 891623 (widening) and 900660 (detection); lane B alone 897023; lane B merged with lane A 900584, 584 over its declaration; lane C alone 905106 and 914143, over both. The remedy the eval names (re-measure by dry run, move the declaration to the smallest ten-thousand boundary with at least one per cent headroom, restamp measured_tokens_est, measured_bytes and measured_at, commit with the reason) has now been run by hand four times (twice on 2026-09-15, precedent fc5c7e27; twice in the pilot), each time by a fresh agent at about fifteen minutes and eighty thousand tokens. Wanted: a verb (working name: reading calibrate) that performs the stated rule for every position and prints the commit body, so a merged tip is recalibrated by one command; the smaller rung is a script under scripts/. Separately for the product thinker: whether one per cent is the right margin when a single lane's text exceeds it.
