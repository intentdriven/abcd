---
schema_version: 1
id: "iss-2608211142146585"
slug: "release-gate-runbook-verify-job-platform"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: ".abcd/development/release-gate/README.md"
resolution: "Corrected the runbook: the verify job now says it runs on Linux (ubuntu-latest) with the macOS+Linux matrix attributed to ci.yml's check job, and 'The verify job gates the merge' corrected to ci.yml's required checks gating the merge with verify re-running the gates against the tagged commit before publish. Docs-only; gate_lockstep's numbered gate list untouched."
impact: internal
---

The release-gate runbook (.abcd/development/release-gate/README.md) misdescribes release.yml's verify job in two present-tense claims a maintainer reads at tag time. (1) It says the verify job runs the nine gates 'on macOS + Linux', but release.yml's verify job is runs-on: ubuntu-latest with no matrix — Linux only; the macOS coverage is ci.yml's check job. The gate_lockstep record-lint rule mirrors only the verify STEP NAMES, never runs-on, so the platform claim is outside its reach. (2) 'The verify job gates the merge' is false: verify runs post-merge on the tag/workflow_call; ci.yml's required checks gate the merge. Doc-accuracy family of resolved iss-182.