---
schema_version: 1
id: "iss-2609231159264048"
slug: "testbannerrendersaboveboard-renders-the-bare-board-twice-in"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/banner_test.go"
resolution: "The banner test renders its board in a temporary directory, where no sibling worktree is read, so a peer session can no longer change the bytes between its two renders."
impact: internal
resolved_by:
  commit: "5be64cdbe4729ef6ab17a260788089a748fc5c49"
---

TestBannerRendersAboveBoard renders the bare board twice in the package directory of the live checkout and requires the two renders to agree byte for byte; since the board gained its peers line it reads the sibling worktrees on disk, so a peer session creating or retiring a worktree between the two renders changes the live-peer count and fails the test (seen in a preflight: 30 live peers without the banner, 31 with it). The test is not hermetic: it should render in a directory with no peers.

## Grounds

- pursued: the test passes whatever peer worktrees exist beside the checkout; a preflight failing it with differing live-peer counts in the two renders would show it wrong
