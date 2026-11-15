---
schema_version: 1
id: "iss-2610230851446558"
slug: "17-guard-md-says-a-broken-guard-never"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "State the legacy-folder exception in the guard chapter's fail-open paragraph, with its reason and the rename that clears it."
resolution: "The guard chapter's Fail-open-loud opening paragraph now states the old-home exception: while anything stands at ~/.abcd the hook exits 2 on every call, because nothing may run unguarded while abcd reads neither folder, and `mv ~/.abcd ~/.abcd.noindex` clears it (in a plain terminal when no plugin-root binary exists or both folders stand). The detector test is not on the branch: widening the hold to it needed a push the sandbox could not authenticate, so the task needed the sandbox lifted; the test is kept in the lane directory. make preflight was cut off at the race gate; make precheck ran clean."
impact: fix
resolved_by:
  commit: "6122cc19b1d49a08aec29208ccbf664f2f8d31f0"
---

17-guard.md says a broken guard never bricks a session, but hooks/pre-tool-use.sh exits 2 (blocking) on every shell and question call while a legacy ~/.abcd folder exists, until it is moved.

## Grounds

- pursued: a reader of the Fail-open-loud section learns that an old ~/.abcd blocks every guarded call, why, and the rename that clears it; shown wrong if the section's first paragraph lacks the exception, or if the hook does not exit 2 in that state (hooks/pre-tool-use.sh, renderHomeStop).
