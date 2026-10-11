---
schema_version: 1
id: "iss-2608211142496517"
slug: "scaffold-sync-doc-comment-names-a"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "cmd/scaffold-sync/main.go"
resolution: "Rewrote the package doc comment to match Makefile:72-76 and iss-209: run by hand via make scaffold-sync, nothing in CI invokes it, drift gated by TestSelfScaffoldParity/TestSyncRepoPinsIsCleanToday, workflow_run propagation a recorded dead end."
impact: internal
---

cmd/scaffold-sync/main.go's package doc comment states the command is 'run by the scaffold-sync workflow on a dependabot branch', but no such workflow exists or ever existed: the string appears nowhere in .github/, and iss-209 records that a workflow_run automation was built and rejected in review (a GITHUB_TOKEN push raises no events, leaving the PR permanently pending; and it was unsound). Makefile:72-76 states the opposite outright ('Nothing in CI calls either target'). The false claim misdirects whoever debugs a red TestSelfScaffoldParity to wait for automation that will never run or re-attempt the dead end iss-209 exists to stop.