---
schema_version: 1
id: "iss-2610230643562468"
slug: "github-pushers-passes-over-an-activity"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "release-hold round 4 (iss-2610222014550692)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/github/activity.go"
remedy: "Have github.Pushers report how many push entries it passed over, and treat any as a record that settles nothing in the release hold's placement"
---

github.Pushers passes over an activity entry with no actor or a malformed field without saying so. The release hold now requires the record's pushes to run unbroken down to the branch's first push, which catches a missing link, but not two missing entries that cancel out (a push and the rewind over it); only a count of the entries passed over closes that.
