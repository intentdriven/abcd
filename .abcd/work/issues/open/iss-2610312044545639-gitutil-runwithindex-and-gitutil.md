---
schema_version: 1
id: "iss-2610312044545639"
slug: "gitutil-runwithindex-and-gitutil"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/index.go"
remedy: "Run every command checkoutDigest takes under a bounded deadline, as internal/gitutil/bounded.go does: pickGitIndex's add -A first, since it runs content filters, then RunWithIndex and StreamWithIndex; fold a timed-out run into indexDigest's uncopied value, or into a hand-back naming the stalled command, so the lane is handed back."
---

The auto-plan's checkout digest runs its git commands with no deadline: gitutil.RunWithIndex and gitutil.StreamWithIndex (internal/gitutil/index.go) and, above all, checkoutDigest's `add -A` through pickGitIndex (internal/core/implement/loop/autoplan.go about line 207, pickcommit.go about line 167), which runs the person's configured content filters. Unlike the bounded runner in internal/gitutil/bounded.go, a stalled filter or git holds the digest, and the implement step taking it, with no limit.
