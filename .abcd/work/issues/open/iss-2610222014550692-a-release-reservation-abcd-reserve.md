---
schema_version: 1
id: "iss-2610222014550692"
slug: "a-release-reservation-abcd-reserve"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "PT asked to close the gap during the 2026-10-22 crash recovery, after the release-hold explanation"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reserve.md"
remedy: "when abcd reserve takes a release, it also withdraws every merge already armed or queued on the default branch (GitHub GraphQL disablePullRequestAutoMerge and dequeuePullRequest), records each withdrawn pull request on the reservation, lets the cutting session keep any it rules in, and re-arms the rest automatically once the release tag reaches origin; the launch checks then refuse a cut while any merge the reservation does not account for is armed"
---

A release reservation (abcd reserve rel-vX.Y.Z) holds only merges armed after it: a pull request whose auto-merge was already on, or that already sat in the merge queue, still merges into the default branch mid-cut, from any copy or account, a running drain's included. Today the cutting session has to list and rule on those by hand right after reserving and again before merging the release pull request (commands/reserve.md, /abcd:launch 'Before step 1'), a manual step that is easy to miss on an every-other-day cadence.
