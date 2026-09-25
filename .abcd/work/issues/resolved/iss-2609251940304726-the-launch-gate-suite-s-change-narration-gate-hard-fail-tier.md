---
schema_version: 1
id: "iss-2609251940304726"
slug: "the-launch-gate-suite-s-change-narration-gate-hard-fail-tier"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
resolution: "The narration hard tier refuses genuine narration again: every construct is refused except its present-state readings (comparative and is/are relative-clause no longer, the is-renamed-to-match purpose form, previously/now with no clause-opening previously and no change verb within three words, used to as a passive or participle); the must-refuse and must-pass sets are one table, narrationSpecification, read by TestNarrationGateHardFailsOnAChangeConstruct and TestNarrationGatePassesPresentTense; the residual trade is in DECISIONS.md (2026-09-25)."
impact: fix
resolved_by:
  commit: "45ace81f"
---

The launch gate suite's change-narration gate (hard-fail tier) misses genuine narration after its round-1 narrowing: 'Previously, the ledger was a flat file; now it is a folder.' passes because changeVerbBesideEither counts ',' and 'the' as tokens, so 'was' falls outside its three-word window (itd-65 AC3 names 'previously X, now Y' as the hard-fail case); 'The registry can no longer be edited by hand.' and 'The scanner no longer skips fenced blocks.' pass because 'no longer' fires only beside a fixed list of subject nouns (AC10 says a genuine 'no longer' DOES hard-fail); 'It was renamed to abcd lint.' passes for the same reason; and 'Until v0.6 the dry-run used to skip the tags.' passes because pastHabitUsedTo requires the subject phrase to open its clause. Docs lint has no token for 'used to' or 'renamed to', so the last two shapes are caught by neither tier. Remedy: count word tokens only in the previously/now window, restore 'no longer' and 'renamed to' as hard fails exempting only the relative-clause and comparative forms ('files that are no longer present', 'no longer than') and the purpose form ('is renamed to match'), and read a past-habit 'used to' after a time adverbial.

## Grounds

- pursued: every sentence in narrationSpecification marked refuse hard-fails and every one marked pass passes, including round 1's four false positives, and the shipped doc bodies carry 0 findings; a narrationSpecification refusal passing, a pass refused, or a finding in the 15 shipped doc bodies would show it wrong
