---
schema_version: 1
id: "iss-2608310912206749"
slug: "two-specs-state-prose-facts-about-the-tree-that-are-false-in"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "fidelity-audits"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs"
---

two specs state prose facts about the tree that are false including a staging count wrong on all three numbers

Two instances found by the two fidelity audits, filed together because they are
one class and the class is already diagnosed (iss-2608301918362294, itd-195).

**spc-57** states a staging measurement: "10 of the 66 `planned/` records carry
an entry, 56 fail". Measured at integration HEAD: **60 planned, 4 with grounds**.
The auditor measured 63/6/57 at the branch tip. Every number is wrong, and the
spec is inside the branch that filed the diagnosis that prose facts about the
tree cannot be maintained. The class recurs here on a COUNT rather than an
enumeration, which is worth noting because a count looks more like a fact and is
no more checkable.

**spc-67** states that the surprise record "is declared once, in spc-58's
family" and that spc-67 "declares nothing a second time". It declares the
family, the store, the required set and the allow-list itself; spc-58 reserves
only a dormant `occasioned_by` key on the reading envelope. The code comment at
the declaration is honest; the spec prose is not.

Both are the shape itd-195 covers, and both argue the same thing it does: the
remedy is to stop stating such facts in prose, not to correct them again.
