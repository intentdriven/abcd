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
---

The launch gate suite's change-narration gate (hard-fail tier) misses genuine narration after its round-1 narrowing: 'Previously, the ledger was a flat file; now it is a folder.' passes because changeVerbBesideEither counts ',' and 'the' as tokens, so 'was' falls outside its three-word window (itd-65 AC3 names 'previously X, now Y' as the hard-fail case); 'The registry can no longer be edited by hand.' and 'The scanner no longer skips fenced blocks.' pass because 'no longer' fires only beside a fixed list of subject nouns (AC10 says a genuine 'no longer' DOES hard-fail); 'It was renamed to abcd lint.' passes for the same reason; and 'Until v0.6 the dry-run used to skip the tags.' passes because pastHabitUsedTo requires the subject phrase to open its clause. Docs lint has no token for 'used to' or 'renamed to', so the last two shapes are caught by neither tier. Remedy: count word tokens only in the previously/now window, restore 'no longer' and 'renamed to' as hard fails exempting only the relative-clause and comparative forms ('files that are no longer present', 'no longer than') and the purpose form ('is renamed to match'), and read a past-habit 'used to' after a time adverbial.
