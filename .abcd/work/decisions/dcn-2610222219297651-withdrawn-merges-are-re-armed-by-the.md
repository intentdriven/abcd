---
id: dcn-2610222219297651
slug: withdrawn-merges-are-re-armed-by-the
date: 2026-10-22
---
2026-10-22 — Withdrawn merges are re-armed by the cutting copy when it ends the release reservation (on --release, or on its next abcd act once the tag is on origin), and as a fallback by any copy whose holder is the same person as the reservation's Abcd-Holder, on any machine or account of theirs, when its reservation fetch sees an ended release with withdrawals not yet re-armed; it writes a tombstone, and re-arming is idempotent. A copy of another person never re-arms: it reports the stranded withdrawals on its peers line and in a tombstone note naming the holder (iss-2610222014550692, choice 2A as amended). Grounds: a cut that dies mid-way, as on 2026-10-18, would otherwise strand every withdrawn pull request with nothing pointing at it; and re-arming runs under the re-armer's own GitHub identity, so a second contributor's machine would otherwise arm someone else's pull requests in its own name. Decided by the coordinating session on the product thinker's delegation while the product thinker is away, amending the technical facilitator's recommendation.
