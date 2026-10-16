---
schema_version: 1
id: "iss-2610160039419977"
slug: "the-cloud-collect-s-tooling-rule-folds"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "review of spc-2610120941441282 fix round 3, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/collect.sh"
remedy: "Refuse any collected path holding a non-ASCII character under a tooling directory, or compare paths through the file system's own case folding, with a test on a case-insensitive volume"
---

The cloud collect's tooling rule folds case for ASCII only, so on a case-insensitive disk a collected path spelled with a Unicode character that folds to an ASCII letter (the Kelvin sign for k, for example) could land on a tooling path the rule refuses in its ASCII spelling

The route exists only through abcd's own collect script, which copies cloud-authored commits onto the person's computer and runs their preflight; without abcd there is no such path, so under abcd-fixes-what-it-adds abcd fixes it.
