---
schema_version: 1
id: "iss-2609302306019245"
slug: "the-guard-s-foldparents-comment-internal"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "v0.12.0 release gate docs review (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Decide the reading once: either fold a trailing . after a leading .. so ../. warns like .. (and widen 17-guard.md:426 back), or narrow the comment to the forms that fold; in both cases add a guard test that names ../., ./../. and ../../. with their verdicts."
resolution: "The behaviour was made to match the comment: cleanSeparators now hands a path ending in /. to foldParents, so ../. reads as .. and warns on rm-rf-working-directory like ./../. does, and ../../. reads as ../..; the chapter sentence in 17-guard.md names ../. again. TestTrailingDotAfterAParentFolds names ../., ./../. and ../../. with their verdicts. The match is additive (the written form is still compared), so no form that warned or blocked before is allowed now."
impact: fix
resolved_by:
  commit: "d8f230dc3"
---

The guard's foldParents comment (internal/core/guard/match.go:883-885) says a trailing . after a .. is folded, but rm -rf ../. is allowed: the fold of a trailing . happens only past a first segment (./../., $PWD/../. and x/../../. warn on rm-rf-working-directory), while ../., ../../., ./. and /. all allow. Harmless in practice, since rm refuses a path whose last segment is . or .., but the comment promises a reading the code does not make and no test in internal/core/guard names ../. at all. Found by the v0.12.0 release-gate docs review of 4d634c4f0 (a HOLD on 17-guard.md:426, whose sentence c8ddb0425 narrowed to ./../.); pre-existing from 4ef5013bd.

## Grounds

- pursued: a recursive delete of the parent spelt with a trailing dot warns as the bare parent does; a guard check of that form returning allow, or a written-form match the fold displaced, would show it wrong
