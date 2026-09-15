---
schema_version: 1
id: "iss-2608301803425790"
slug: "five-nits-from-the-itd-179-delta-review-including-two-export"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-delta-ruthless"
found_at: "internal/core/mdrecord"
resolution: "all seven are settled: the claims.go citation now names IsTopLevelBullet and BulletBlocks for the rules they actually carry, IsAnyBullet is deleted and Masked unexported, the triage verbs pass the verb rather than the target state, the writer refuses an ambiguous second Grounds heading as the intent half already does, groundsEntries says which reader it asks, mdrecord has its own tests, and the backtick scan is measured and deliberately left"
impact: internal
resolved_by:
  intent: "itd-179"
---

five nits from the itd-179 delta review including two exported functions with no caller and a renamed citation that now names the wrong predicate

Five nits from the itd-179 delta review, held on one record. Per the round rule
adopted 2026-08-30 these are settled at the ship commit, not by re-opening a
review.

1. `intent/claims.go` -- the extraction renamed a correct citation into an
   incorrect one. It reads `mdrecord.IsAnyBullet already rules for acceptance
   criteria`; pre-extraction it cited the TOP-LEVEL predicate, and acceptance
   criteria are counted by `IsTopLevelBullet`. `IsAnyBullet` rules nothing about
   them.
2. `mdrecord.go` -- `IsAnyBullet` is exported with no caller in the tree (its
   only occurrence is the mis-citation above), and `Masked` is exported with no
   caller outside its own package.
3. `capture/workflow.go` -- `appendGrounds` is passed the STATE rather than the
   verb, so one command emits two prefixes: `resolve: grounds refused` from the
   flag check and `resolved: grounds refused` from the append.
4. `grounds/record.go` -- a record with two `## Grounds` headings reads only the
   first, so the second section's entries are invisible everywhere. The intent
   half guards the analogous case for scope conditions by refusing to stamp when
   it counts more than one heading; grounds has no such guard on either side.
   Pre-existing pattern on new surface.
5. `capture/validate.go` -- `groundsEntries`'s comment says it asks the same
   reader the intent half asks. The intent half asks `ParseSectionAboveFloor`;
   this asks `ParseSection`. The next paragraph corrects it, so the block
   contradicts itself rather than being simply wrong.

Two more from the delta SECURITY review, folded here rather than given ids:

6. `internal/core/mdrecord/` ships with no test files of its own and is covered
   only transitively through `core/intent` and `core/grounds`. A new leaf in the
   core DAG with no direct tests is the shape adr-57 makes load-bearing.
7. `opensCommentFrom`/`findBacktickRun` is superlinear on a line carrying many
   distinct-length backtick runs, because each unmatched run rescans to end of
   line. PRE-EXISTING, moved verbatim from `claims.go`; record files are small,
   so there is no practical consequence today.

## Grounds

- pursued: we expect an unused export and a mis-cited predicate to mislead the next reader in the same way a false message does, so both are settled at the ship commit rather than carried
