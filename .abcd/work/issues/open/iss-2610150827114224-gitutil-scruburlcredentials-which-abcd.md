---
schema_version: 1
id: "iss-2610150827114224"
slug: "gitutil-scruburlcredentials-which-abcd"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "abcd land review round 3, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/userinfo.go"
remedy: "Make gitutil.ScrubURLCredentials parse each URL-shaped run with net/url instead of a regex, so a password holding a raw slash is still stripped and https://host?q=a@b is not rewritten; add table tests for both"
---

gitutil.ScrubURLCredentials, which abcd land and the build loop apply to git and forge text they print, finds URLs with a regex whose authority ends at the first slash: a password containing a raw slash (https://u:p/w@host/x) is left unscrubbed, and a query holding an at sign (https://host?q=a@b) is over-redacted to https://b. Git percent-encodes such characters in URLs it prints, so the leak needs a hand-written URL, and over-redaction only garbles a message. Route exists without abcd: no, the scrubber is abcd's, so abcd fixes it. Found by the round 3 reviewers of the abcd land change, 2026-10-15.
