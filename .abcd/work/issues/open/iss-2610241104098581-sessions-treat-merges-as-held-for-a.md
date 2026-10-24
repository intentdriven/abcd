---
schema_version: 1
id: "iss-2610241104098581"
slug: "sessions-treat-merges-as-held-for-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "abcd build itd-82 session, 2026-10-24, waiting on the v0.15.2 hold"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/release.go"
remedy: "abcd peers and abcd status name a release reservation that expired with no tag on origin as a lapsed cut holding no merge, with its holder, its last renewal and that a cut resumed from any copy reserves the release again (lapsed_releases in --json); the landing already reads only a live release reservation as a hold, and re-reading a hold before relaying it, and lifting a lapsed one unattended, stay with iss-2610241107191205"
---

Sessions treat merges as held for a release cut whose holder has crashed: on 2026-10-24 local session abcd-0d told every session that v0.15.2 is being cut so merges are held, while the only marker of that cut on origin is the reservation rel-v0.15.2, held by copy 07cd3b4c167cec332f8c01cf42ad41d6 (a computer that crashed), last renewed 2026-10-23T08:52:49Z and expired 2026-10-24T08:52:49Z, with no standing announcement, no release branch and no newer tag. Nothing names a cut as abandoned: abcd status lists rel-v0.15.2 as an ordinary expired reservation, so a session reading it, or relaying a hold heard earlier, keeps work waiting on a session that no longer exists, and two finished pull requests (the drain spec's landed marks, the fix for iss-2610110918418459) sat waiting for a ruling from it.
