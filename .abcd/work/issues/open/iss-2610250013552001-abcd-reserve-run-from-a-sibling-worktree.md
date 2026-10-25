---
schema_version: 1
id: "iss-2610250013552001"
slug: "abcd-reserve-run-from-a-sibling-worktree"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run, renewing reservations 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/remote.go"
remedy: "When a tip's copy key is not this copy's, look it up among the checkout's other worktrees (git worktree list, each one's .abcd/.work.local/reservation/copy-key, read and never minted) and name that worktree, home-redacted, in the reserve refusal and the release merge hold's reason and waits ('from your worktree <path> on this machine'), keeping 'another computer' for a key no worktree holds; and have the footprint refusal say a renewal names it too."
---

abcd reserve, run from a sibling worktree of the checkout that holds a person's live reservation, refuses with 'you are already building <id> on another computer', and the release merge hold reads 'being cut by your copy on another computer' the same way. The copy key lives in each worktree's own local tier, so every worktree is its own copy and the words are false for the commonest case, a second worktree on the same machine: on 2026-10-25 a renewal from the lane's worktree was refused that way, and only trying the main checkout found the copy that held it. The footprint refusal does not say that a renewal by hand, the same command again, names the footprint too.
