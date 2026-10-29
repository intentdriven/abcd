---
schema_version: 1
id: "iss-2610252214566448"
slug: "02-constraints-01-platform-md-21-says-a"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S10"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md"
remedy: "Replace the two clauses in 01-platform.md with pointers to 02-disembark.md § 7 and 03-embark.md § 7."
refines: [iss-2609091956001547]
---

02-constraints/01-platform.md:21 says 'a test hashes its tree before and after' and :23 says past disembarks are recorded as 'hash + file list + label'; the test is a path/size/mode fingerprint (internal/surface/cli/disembark_test.go:240-258; 02-disembark.md:234-235 'a fingerprint, not a content hash') and the voyage entry carries no file list and no label (internal/core/lifeboat/voyage.go:27-43; 03-embark.md:300-303).
