---
schema_version: 1
id: "iss-2610110359273529"
slug: "record-id-redraw-differs-by-family"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "abcd-60 drain run 2026-10-11, lane for iss-2610092011131598"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture"
remedy: "Make mintRecordID redraw on any existing path (bounded, as the iss and reading-item allocators do under spc-33 ruling 2) and drop the in-process minted-path set, keeping refuseExistingRecord as the write-time guard; test a forced clash with a planted foreign record, watched fail first."
resolution: "mintRecordID now redraws on any existing record path within placeholderRetryBudget, as the iss and reading-item allocators do (spc-33 ruling 2); the in-process minted-path set is gone and refuseExistingRecord stays the write-time guard."
impact: fix
resolved_by:
  commit: "e76a0da22f749fd63d6b24452321095eddc52c3e"
---

After the drain lane for iss-2610092011131598 lands, the surprise, admission, disposition and reframe records redraw a minted id only when it clashes with a record this same process wrote, and still refuse a clash with any other existing record. The issue and reading-item allocators redraw on any clash (spc-33 ruling 2). Both of that lane's reviewers noted the split. The retained refusal guards against overwriting, which a redraw never does, so it protects nothing a redraw would endanger, and it can still fail a same-second mint against a peer's committed record.

## Grounds

- pursued: a surprise, admission, disposition or reframe mint whose draw lands on a peer's committed record from the same second now lands on a fresh id and leaves the peer's record untouched (TestSurpriseRedrawsPastAPeersRecord, TestMintRecordIDRedrawsOnAnyTakenPathWithinTheBudget); a mint refused with ErrDuplicateIssueID on such a clash, or a peer record overwritten, would show it wrong.
