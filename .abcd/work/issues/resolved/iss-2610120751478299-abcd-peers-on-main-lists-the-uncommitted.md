---
schema_version: 1
id: "iss-2610120751478299"
slug: "abcd-peers-on-main-lists-the-uncommitted"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "embargo steps 10-11 lane, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/peers"
remedy: "Skip the embargo store in abcd peers unread, counting it without a path, and keep private records only in the store once abcd embargo ships, so no sibling worktree peers reads holds one."
resolution: "abcd peers skips the embargo store unread and counts it with no path (commit 6c8dc0a9e, test TestPeersSkipsEmbargoStore in internal/core/peers/peers_test.go); once abcd embargo ships, private records live only in that store, so no sibling worktree peers reads holds one"
impact: fix
resolved_by:
  spec: "spc-2610120557432880"
  commit: "6c8dc0a9eb94d9b7680a2b729d14361cee07b8e5"
---

abcd peers, on main, lists the uncommitted records it finds in every sibling worktree of the checkout, titles included. A private record held in a hand-run private worktree, or in the embargo store, is therefore shown to every local session that runs abcd peers, and a title alone can disclose a security report still under embargo. Under the security scope rule abcd adds that route itself, so the leak is abcd's to close. Found 2026-10-12 by the embargo steps 10-11 lane.
