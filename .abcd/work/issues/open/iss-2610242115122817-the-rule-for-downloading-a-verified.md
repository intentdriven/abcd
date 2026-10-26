---
schema_version: 1
id: "iss-2610242115122817"
slug: "the-rule-for-downloading-a-verified"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update"
remedy: "Extract one download-and-verify primitive (allowed hosts, redirect policy, size cap, binding to the release tag and checksum) into internal/core/update or a small shared package, and have the tool installer's fetch call it, with a test that both callers refuse the same off-host redirect."
---

The rule for downloading a verified release now exists twice: abcd's own update path (internal/core/update) and the new tool installer on the setup redesign branch each decide which hosts may be reached and how a download is bound to its release, so any fix to that rule must be made in both and can drift between them.
