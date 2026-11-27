---
id: dcn-2611270714216273
slug: a-preflight-failure-on-a-lane-whose
date: 2026-11-27
---
2026-11-27 — A preflight failure on a lane whose review fix rounds are spent is fixed when the fix is mechanical and changes no behaviour, and it does not count as a review round. The matrix globs lane (six rounds, all spent) failed preflight on two existing gate tests: an exitError literal in lab_matrix_launch.go outside refused(), and a commands/lab.md question that names no role. Both are fixed to the gate's form and re-preflighted, without re-review beyond a delta stamp. Grounds: the person's round grant bounds review churn, not the act of passing the gates the change must meet anyway. Reverse by holding the lane back for the person instead.
