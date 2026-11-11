---
schema_version: 1
id: "iss-2611101318042677"
slug: "nothing-checks-that-the-commit-a-spec-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/specstepsserial.go"
remedy: "Have record-lint refuse a landed value naming a commit the checked range cannot reach, as RS003 does for resolutions, accepting a pull request number as it stands; a test rewrites a landed commit away and finds record-lint refusing."
---

Nothing checks that the commit a spec's landed line names is reachable from the branch. On 9 Nov two commit subjects on a lane were reworded with filter-branch, and the spec's step-4 landed line still named the commit from before the rewrite, 0c9d9721a, which the branch no longer reached; a reviewer caught it and the fix round repointed it to 904e2a8f5. RS003 makes the same check for an issue's resolved_by.commit.
