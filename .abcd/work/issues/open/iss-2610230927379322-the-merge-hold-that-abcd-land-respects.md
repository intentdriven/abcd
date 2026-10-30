---
schema_version: 1
id: "iss-2610230927379322"
slug: "the-merge-hold-that-abcd-land-respects"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "release hold round 5 (iss-2610222014550692)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/release.go"
remedy: "Give releaseHolds the same ended reading as asCut (a tombstone ends a hold only when the record of pushes places it on the holder, or on its tag), with a cached per-tip answer so a land check does not ask the forge each time"
---

The merge hold that abcd land respects (claim.Hold, read by releaseHolds) still reads a release as ended as soon as any tombstone sits on its reservation branch, whoever pushed it. The release hold now ends a release only on a tombstone its holder pushed or on its tag, for the second-release refusal, the launch reading and peers, so the two readings of one release can disagree: land lets merges through while the launch check still names the release as being cut. The land hold already ends when its lease lapses, so it is a time-boxed courtesy and the withdrawal is the hold that keeps merges out; the disagreement is in what each surface reports.
