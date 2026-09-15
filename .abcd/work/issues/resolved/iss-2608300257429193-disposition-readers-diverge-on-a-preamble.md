---
schema_version: 1
id: "iss-2608300257429193"
slug: "disposition-readers-diverge-on-a-preamble"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-180 second-round review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go (dispositionsOf), internal/core/capture/reading.go (standingDispositions)"
resolution: "The standing-answer judgement moved into the issueschema leaf that both packages call, so a preamble-led or duplicate-keyed record retires nothing on either side; lint.StandingDispositions is gone and the parity table gained comment-led and blank-line-led rows."
impact: fix
---

The two standing-disposition readers still diverge: lint's frontmatterFields tolerates a preamble before the opening --- and reads the supersedes edge, while capture's parser requires --- on line 0, errors, and leaves both dispositions standing, so a comment-led or blank-line-led disposition file makes the board say answered and the verb say two standing (promote refuses). Refuse a preamble-led file in dispositionsOf as the duplicate-key branch does, or converge both packages on one reader in the issueschema leaf; extend the parity table with both shapes.
