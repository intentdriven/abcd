---
schema_version: 1
id: "iss-2609300711394491"
slug: "the-drain-rule-offer-shifts-a-piped-ahoy"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/drain_rule.go"
remedy: "Ask the drain-rule category and offer only when the prompter is at a terminal (the itd-131 TerminalPrompter precedent the git identity step set), neither asking nor counting it declined off one, and list drain_rule.offered under optional_skipped naming the terminal as the way to be asked; test that a piped stream of the pre-change answer count gets the pre-change questions and writes no record."
resolution: "Resolved: the drain-rule category and offer are asked only at a terminal; off one they are neither asked nor counted declined and drain_rule.offered is listed under optional_skipped. TestAPipedInstallStreamIsNotShiftedByTheDrainRuleOffer."
impact: fix
resolved_by:
  commit: "6683d62c8"
---

The drain-rule offer shifts a piped ahoy install answer stream: the drain-rule category question and the offer itself are inserted mid-order, and the stdin prompter reads one line per question with no terminal gate, so in every adopter repository a scripted stream hands the answer meant for a later question (user-state, the identity pin) to the drain rule, can write an accepted drain eligibility record the script never asked for, and leaves the later question reading EOF.

## Grounds

- pursued: a piped stream of the pre-change answer count gets the same questions in the same order and writes no record; a drain-rule question put to a pipe would show it wrong
