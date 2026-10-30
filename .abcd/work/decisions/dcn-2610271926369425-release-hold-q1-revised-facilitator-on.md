---
id: dcn-2610271926369425
slug: release-hold-q1-revised-facilitator-on
date: 2026-10-27
---
2026-10-27 — Release hold, Q1 revised (facilitator on delegation, lane D on abcd-f5's ruling, 2026-10-27; supersedes Q1 of dcn-2610230328220298): a release reservation whose lease lapses with no release tag on origin ends its hold. The merges this cut recorded as withdrawn are re-armed, and only those, each with one comment on its pull request saying the cut lapsed. A merge the cut did not record is never armed. A cut that outlasts its lease renews it with abcd reserve rel-vX.Y.Z, so a live cut's withdrawals stay withdrawn. Grounds: on 2026-10-24 the person ruled that sessions do not hold merges for a lapsed release cut, which #1132 implements by naming a lapsed cut as holding no merge (iss-2610241104098581). That day two finished pull requests waited on rel-v0.15.2, a cut whose computer had crashed. Under the old Q1, merges such a cut withdrew stay withdrawn indefinitely, which contradicts that ruling. Q1's own risk, a cutter's next act re-arming merges mid-cut, is met by renewal: a cut that is still running is still renewing.
