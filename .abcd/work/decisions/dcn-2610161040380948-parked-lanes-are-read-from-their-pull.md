---
id: dcn-2610161040380948
slug: parked-lanes-are-read-from-their-pull
date: 2026-10-16
---
2026-10-16 — Parked lanes are read from their pull requests (facilitator-on-delegation, drain-many-accounts): each drain move reads every parked lane's pull request in one forge listing; a closed pull request releases its lane's reservation and hands the lane back, a conflicting one stops being renewed and is named in next, and a parked lane whose head is not yet on the default branch is named as waiting rather than skipped (grounds: the pull request's state is the evidence of the lane's progress, and one listing costs one request a move; iss-2610161015381750).
