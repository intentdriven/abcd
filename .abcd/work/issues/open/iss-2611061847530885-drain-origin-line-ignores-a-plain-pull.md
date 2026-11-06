---
schema_version: 1
id: "iss-2611061847530885"
slug: "drain-origin-line-ignores-a-plain-pull"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "first real drain dry run ahead of a long run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Read the age from the remote-tracking ref's own reflog or FETCH_HEAD's modification time, whichever is newer, rather than a stamp only abcd writes."
---

The origin line of abcd drain --dry-run says when the checkout last fetched by reading only abcd's own fetch stamp, so straight after a git pull it reported the last fetch as 107 minutes old while also, correctly, reporting no commits behind. A reader takes that as a stale plan when it is not.
