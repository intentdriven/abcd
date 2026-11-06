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
found_at: "internal/core/implement/loop/drain_origin.go"
remedy: "Read the age from the newer of abcd's stamp and FETCH_HEAD's modification time, counting FETCH_HEAD only when it names the default branch, since a fetch of another branch alone says nothing of how fresh the default branch is."
---

The origin line of abcd drain --dry-run says when the checkout last fetched by reading only abcd's own fetch stamp, so straight after a git pull it reported the last fetch as 107 minutes old while also, correctly, reporting no commits behind. A reader takes that as a stale plan when it is not.
