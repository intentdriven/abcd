---
schema_version: 1
id: "iss-2609091732329046"
slug: "closing-a-spec-moves-its-intent-but-leaves-every-link-that-n"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "closing three specs after the sub-agent capture work"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec"
resolution: "The record-moving verbs (spec close, intent plan, capture resolve, capture wontfix) repoint every relative link that named the moved record's old path through one primitive, core/relink, and report each rewrite."
impact: fix
resolved_by:
  commit: "2a6d5b63"
---

Closing a spec moves its intent but leaves every link that named the intent's old folder pointing at nothing. The close verb reconciles the intent from planned to shipped, which is its job, and the spec body that was written while the intent was planned keeps its relative links to the planned folder. Those links resolve to nothing the moment the move completes, and the record gate refuses on them, so a close that reports success hands the next command a tree that will not lint. Three closes in one sitting produced eight dead links here and a red preflight immediately afterwards, with nothing in the close output hinting at it. The verb already knows both the old and the new path, so it is the one thing in the system positioned to fix or at least name them. Either rewrite links to the moved record in the same operation, or refuse the close while a link in the spec names the folder the intent is about to leave, or say at minimum which links the move has just invalidated. Silence is the worst of the three, because the failure surfaces later, in a different command, as a lint error that looks unrelated to the close that caused it.

## Grounds

- pursued: we expect a spec close, plan, resolve or wontfix to leave a tree record-lint's links_resolve accepts with no hand repair; a links_resolve blocker naming a moved record's old path after one of those verbs would show it wrong
