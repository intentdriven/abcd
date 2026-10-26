---
schema_version: 1
id: "iss-2610252224180535"
slug: "the-header-comment-of-github-workflows"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "state-of-the-art passes of the hand-back triage, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Change the header to RS001-RS006, or drop the range from the header and leave it to the job's own comment at ci.yml:711."
---

The header comment of .github/workflows/ci.yml (line 19) says the issue-resolution gates are RS001-RS003, while the job it describes runs RS001 to RS006 (ci.yml:711-751).
