---
schema_version: 1
id: "iss-153"
slug: "privacy-hygiene-abspathre-flags-users"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "prepare-this-repo run against an external repo"
found_at: "internal/core/audit/rule_privacy.go"
resolution: "absPathRe exempts the macOS system directories under /Users, so product code needs no per-line waiver"
impact: fix
---

privacy-hygiene absPathRe flags /Users/Shared/... (and would flag /Users/Guest/...) — macOS system directories, not usernames. Exempt well-known non-user segments after /Users/ so product code that legitimately uses /Users/Shared does not need per-line waivers.