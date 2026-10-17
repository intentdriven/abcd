---
schema_version: 1
id: "iss-2610152255268094"
slug: "the-cloud-report-s-issue-comment"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "build of spc-2610120941441282 step 8, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/forge-repo.sh"
remedy: "Record the hosted clone's origin URL in the spec's live step 12; if it is not github.com, let the host connector declare how the forge repository is found, keeping the strict parse for everything outside it"
---

The cloud report's issue-comment fallback finds the repository by parsing origin's URL strictly as github.com, but a hosted session may clone through a proxy URL, in which case the fallback could never post and a session whose push failed would leave its report only in the transcript
