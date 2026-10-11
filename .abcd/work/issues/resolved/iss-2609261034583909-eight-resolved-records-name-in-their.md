---
schema_version: 1
id: "iss-2609261034583909"
slug: "eight-resolved-records-name-in-their"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lintA item 4"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/resolved"
resolution: "Annotated, not corrected: a resolution note true when written stays as written (RS006 re-reads only records entering a terminal folder, for that reason), and each record whose note names a removed test gains a dated Correction section naming the superseding commit, the successor test and which part of the claim still holds. Six records carry one from this change (iss-184, iss-2608301808193750, iss-2608311632382737, iss-2608311632439831, iss-275, iss-354); iss-2609012039117381 already carried one from 2026-09-09 (2eeaab68a); iss-2609090951291799's mention stays accurate. A ledger-wide sweep found a ninth, iss-2608270559313719, which gains the same section in the commit after this record's resolution. No gate would have caught it, and none should: a terminal record's note is history."
impact: internal
resolved_by:
  commit: "99452aba0"
---

Eight resolved records name, in their resolution notes, a test no longer in the tree; each was true when written, and RS006 reads only records entering a terminal folder, so none is refused, but a reader following the note finds nothing. Verified at b7646ff1, with the commit that removed each test: iss-184 names TestTokenizeRejectsUnterminatedHeredoc (16d50feb renamed it TestTokenizeFlagsUnterminatedHeredoc and changed the behaviour: an unterminated heredoc is a verdict, not ErrUnparsableCommand, so the note's part (2) is stale too); iss-2608301808193750 names TestIsAbsentValueIsASpellingTestNotANullTest (3eb4b549 replaced the spelling predicate with a class-based one, TestAbsenceIsDecidedByClassNotBySpelling, so the note's 'deliberately NOT widened' is superseded); iss-2608311632382737 names TestPreflightRunsBothEvalLanes (the test is TestPreflightRunsEveryTaggedEvalLane, which derives the lanes from the Makefile; the one clean rename of the eight); iss-2608311632439831 names TestComparativeRefusesToAssemble (3b62c967 made the comparative position assemble from the widening run's items, so the refusal the note describes no longer exists); iss-2609012039117381 names TestBinaryHooksFallBackToAPathBinary and iss-275 names TestGuardShimFallsBackToPATH (c637a734 removed the PATH rung those tests pinned: the shims run only the abcd the machine recorded); iss-354 names TestWorkflowGoVersionsMatchSubstitutions (removed in df377ee6, replaced by TestEverySetupGoResolvesTheToolchainFromGoMod under iss-2609090951291799); iss-2609090951291799 names TestWorkflowGoVersionsMatchSubstitutions only to say it was replaced, which stays accurate. Six of the eight need an amendment note stating what superseded the test and the claim, not a rename, so the set is not a mechanical fix.

## Grounds

- pursued: a reader following any of the eight notes to a test that is gone finds, in the same record, the commit that removed it and what pins the behaviour now; a resolved record naming a test absent from the tree with no Correction section saying so would show it wrong
