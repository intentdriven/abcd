---
schema_version: 1
id: "iss-283"
slug: "the-real-name-suppression-breaks-on-an"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/adapter/scanner/identity.go"
resolution: "the suppression also recognises the login embedded in the caller's users.noreply.github.com address; TestIdentityNameEqualsNoreplyLogin pins both directions"
impact: fix
---

The real_name suppression breaks on an org-owned remote: newIdentityMatchers suppresses real_name only when git user.name equals the username parsed from remote.origin.url, so transferring the repo to an organisation (remote owner Partnermedia, user.name REPPL) turns the caller's public GitHub handle into 13 hard_fail findings and blocks the launch. The noreply email (id+login@users.noreply.github.com) carries the caller's login locally and is the right second source for the suppression