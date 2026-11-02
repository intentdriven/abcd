---
schema_version: 1
id: "iss-2611020720003337"
slug: "teststatuslinecreatesnodirectory"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "a landing preflight under load, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_readonly_test.go"
remedy: "Make the subtest assert only the absence of a write notice and of a created directory, tolerating the status line's timed-out facilitator read, or give the subtest a facilitator.mode it can read without git so the row's own timeout never enters it."
---

TestStatuslineCreatesNoDirectory (internal/surface/cli/statusline_readonly_test.go) fails on a loaded machine: its subtest 'an abcd home with no fill directory' takes the status line's 'DROPPED the merging element — facilitator.mode could not be read (git did not answer in time ...: context deadline exceeded)' stderr line for 'a skipped write was said', so a git read that times out under load fails a test about directories. It failed a landing preflight on 2026-11-02 at about 07:15 UTC.
