---
schema_version: 1
id: "iss-2610230643540706"
slug: "the-forge-s-activity-record-is-read-as"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "release-hold round 4 (iss-2610222014550692)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/github/activity.go"
remedy: "Page through the activity record (github.Pushers) until the push that made the branch, within a bound, so a long release branch is read in full"
---

The forge's activity record is read as one page of 100 entries. A release branch with more than 100 activities is read partial for ever, so every withdrawal on it reads not yet verified on every act and is never re-armed by abcd.
