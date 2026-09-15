---
schema_version: 1
id: "iss-2608301519254240"
slug: "four-correct-guards-in-the-outstanding-report-and-the-join-h"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-3-reviews"
found_at: "internal/core/lint/readingoutstanding.go"
resolution: "Guards 1-3 each gain a killing test watched fail under its own mutation: the Unadmitted leg's stand-down (an accepted disposition beside an oversized admission), the unlistable admissions root, and the join's unread-family stand-down, which the test kills as a whole. Its two halves are not separably killable: LoadConfig's validateRecordStores refuses a record_stores key naming a prefix no store declares, so for any configuration a production caller can hold, an unread family is also an unknown one and the !known half is defence against a hand-built Config alone. Guard 4, the unreachable r.bucket == '' in checkRecordBucketField, is deleted rather than tested, and the scope note says why a bucketField-declaring store always has a bucket. The stale 'sameBucket join' comment now says sameBucketAs. The redundant mark(judged, 'slug') stays: mark(judged, 'id') is redundant on identical grounds, so removing one alone reproduces the one-instance-removed shape this record opens with, and removing both leaves judged unused in two signatures against the protocol stated at the call site."
impact: fix
resolved_by:
  intent: "itd-189"
---

four correct guards in the outstanding report and the join have no killing test so each mutation survives the whole suite

Found by the round-3 ruthless review, each proved by a surviving mutation.
Every one is CORRECT code; what is missing is a test that dies without it.

1. `readingoutstanding.go:362` — the Unadmitted leg's stand-down on an
   unreadable run. Delete `!admissions.unknown(run.Name())` and the whole suite
   stays green, while the report then emits "is a widening proposal with neither
   an admission nor a decline" on the strength of a tree nobody read — possibly
   about the very admission that could not be read. The existing stand-down
   tests all use fixtures with NO standing disposition, so they exercise only
   the sibling stand-down at line 337. The reviewer wrote a 12-line probe
   (accepted disposition + oversized admission) that passes unmutated and kills
   the mutation.
2. `readingoutstanding.go:439` — an admissions root that exists but cannot be
   listed sets `rootUnreadable`. Removing it survives: every widening proposal
   in every run would then be judged against a tree nobody listed.
3. `schema.go:839` — the "a family this scan does not read supports no verdict"
   stand-down in `checkRecordJoins`. Removing it survives: `occasioned_by:
   spike-3` would draw a blocker saying it is not a record in the corpus,
   contradicting the leg's own "prose is legitimate and stays silent".
4. `schema.go:903` — `r.bucket == ""` in `checkRecordBucketField` is
   UNREACHABLE (`bucketField` is declared only by the bucketed `adm` store).
   This is the exact twin of the two stand-downs `9d9024da` deleted IN THIS
   ROUND as "a branch no fixture can enter is how a guard comes to look tested"
   — one instance removed, its sibling left.

Why this record exists at all: `7a7dd1ed` was raised in this same round because
the Unadmitted clause was the one guard no mutation could kill. The round fixed
that clause and left three more in the same file, plus an unreachable branch it
had just established a principle against.

Folded here rather than given ids of their own, both cosmetic: `mark(judged,
"slug")` at schema.go:681 is redundant today (no store both requires `slug` and
lacks the issue-shape slug leg, and a blank slug always fails SlugRe), harmless
and arguably right as defence; and readingoutstanding.go:75 still says
"record_schema's sameBucket join" after `9d9024da` renamed the field
`sameBucketAs`.
