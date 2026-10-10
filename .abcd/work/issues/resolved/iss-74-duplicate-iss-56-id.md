---
schema_version: 1
id: "iss-74"
slug: "duplicate-iss-56-id"
severity: "major"
impact: additive
category: "bug"
source: "user-observation"
found_during: "abcd-run-design"
found_at: ".abcd/work/issues/open"
resolution: "Added issue_id_unique record-lint rule scanning open/resolved/wontfix for any iss-N claimed by >=2 files; shares the validateIDUnique primitive with the intent-id check"
---

Duplicate iss-56 id: two open issues share id iss-56 — iss-56-iss-36-lists-abcd-logbook-as-a-retired.md and iss-56-managed-pre-commit-gates.md. The allocator scans max N across open/resolved/wontfix under flock+O_EXCL, so a collision implies a manual add that bypassed it or an allocator gap. Ledger-integrity bug: derived priority, blocked_by edges, and resolve/wontfix by id are all ambiguous while two files answer to iss-56. Detector (per unrecognized-input-never-writes / ledger integrity): a capture/record-lint check that ids are unique across the three status dirs; acceptance corpus = the two iss-56 files. Fix: renumber the later-created one to the next free id and repoint any inbound blocked_by.