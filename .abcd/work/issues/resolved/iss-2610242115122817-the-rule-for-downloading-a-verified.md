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
resolution: "One download rule, urlguard.Download, now carries the allowed hosts, redirect policy, transport pinning and time bound for both the update path and the tool installer, and each purpose (a release download, a provenance check) reaches only its own hosts. The update path's downloads are also held to their hosts on every request, not only on redirects; its latest-release lookup runs through the vintage fetcher, outside this rule. Each caller's own test refuses an off-host redirect through the shared rule, and urlguard's tests pin the rule itself, rather than one test spanning both callers. The size cap stays with each caller, since it differs by what is read, and the binding to the release tag stays in the update path's release check."
impact: fix
---

The rule for downloading a verified release now exists twice: abcd's own update path (internal/core/update) and the new tool installer on the setup redesign branch each decide which hosts may be reached and how a download is bound to its release, so any fix to that rule must be made in both and can drift between them.
