---
schema_version: 1
id: "iss-2611070737220024"
slug: "the-loop-package-s-cached-test-log-is"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil"
remedy: "Cache the resolved home, or skip strings with no path separator."
---

The loop package's cached test log is 800 to 840 MB per run (about 14.8 million lines), mostly from RedactHome resolving home's symlinks about 742,000 times plus PATH lookups; 232 such files hold 51 GB of one account's Go cache.
