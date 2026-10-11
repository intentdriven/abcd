---
schema_version: 1
id: "iss-2609302306153318"
slug: "docs-fidelity-record-and-docs-fidelity"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.12.0 release gate docs review (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/docfidelity/store.go"
remedy: "Validate every failing entry's chapter at record time with the same chapterNameRe Apply uses (or normalise a 04-surfaces/ prefix in both places), so record refuses what apply would refuse; a test records and applies a verdict in each shape."
resolution: "docs fidelity record and --apply now read a brief chapter through one parse (chapterFile): the file name, the name under 04-surfaces/ and the repo-relative path the reviewer's request lists are admitted and saved as the file name, and any other path is refused at record time; TestRecordAndApplyAgreeOnTheChapterShape records and applies a verdict in each shape. The judge-model half (integ24 df2) was already fixed on the base by accf61856."
impact: fix
resolved_by:
  commit: "ba54dbcdc"
---

docs fidelity record and docs fidelity --apply disagree on the shape of a verdict's chapter: record checks only that a failing entry's chapter is non-empty (internal/core/docfidelity/store.go:248-249), so it saved a HOLD naming "04-surfaces/17-guard.md", and --apply then refused that same verdict with "the edit's chapter ... is not a chapter file under .abcd/development/brief/04-surfaces" (Apply's one-component chapterNameRe, internal/core/docfidelity/apply.go:31,64). A verdict the recorder accepts cannot be applied, and the reviewer has to re-record it by hand. Found while applying the v0.12.0 release-gate docs review of 4d634c4f0 (the correction landed as c8ddb0425).

## Grounds

- pursued: every verdict the recorder saves is one the applier can write; a recorded HOLD whose --apply refuses on the chapter, in any shape, would show it wrong
