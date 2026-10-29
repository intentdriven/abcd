---
schema_version: 1
id: "iss-2610161645224876"
slug: "the-messaging-press-release-promises"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "product thinker interview 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/land.go"
remedy: "Make the announcement reach origin before the push it announces: a session that pushes work for an item with a live announcement (or one it is required to make) publishes the announcement first, and the push waits for it. Cover the plain git push path through the pre-push hook as well as abcd land, keep a failed announcement from silently dropping the push's notice, and test that a colleague's status read shows the announcement before the pushed branch exists on origin."
resolution: "abcd land and a build or drain lane's landing now post merging <item> through the announce code and push only once it is on origin, so a colleague's status read shows it before the branch exists there; an announcement that cannot be published refuses the push; a hand-typed git push posts none, and the docs say so"
impact: additive
resolved_by:
  intent: "itd-2609151838312703"
  spec: "spc-2610150728396006"
  commit: "a3e96de9f95efa6d75cd7851d026a4fb59d0a1c0"
---

The messaging press release promises that a colleague's session sees an announcement such as 'merging the login fix' before the session that made it pushes. The build guarantees it only before the work merges into the main line, at the land step. The product thinker chose on 2026-10-16 to make the build keep the promise rather than reword it.

## Design notes 2026-10-16 (from abcd-a9, the messaging-design session, for whoever builds this)

1. **Keep it out of the pre-push hook if possible.** AGENTS.md records why preflight left the pre-push hook: git opens the push connection before the hook runs, and long work there outlasted the transport's idle timeout, so the push reported success and moved nothing. An announcement pushed from inside pre-push adds a second network round trip (a listing and a push) inside that same window. Either keep it short and bounded, or announce before `git push` in the verbs that push.
2. **The hook doesn't cover cloud sessions.** The cloud probe showed the pre-push hook does not run there, so `abcd land` and `abcd announce` must keep the promise, not the hook.
3. **A publishing hook is a new network actor.** Invariant 7 names the verbs allowed to fetch status. A hook that publishes, and so must list or fetch first, needs a clause in the same change, or that path is ruled out. If the record can't settle which, that point goes to the product thinker.
