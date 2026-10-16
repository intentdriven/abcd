---
id: dcn-2607210000000006
slug: a-probe-walk-of-a-foreign-tree-must-be
date: 2026-07-21
---
2026-07-21 — A probe walk of a foreign tree must be bounded in **three**
dimensions, not one. itd-95 shipped `WalkFiles` with a regular-file cap; an
independent security review of the itd-96 branch showed both remaining
dimensions were exploitable — a tree of directories holding no regular file
never reaches a file cap, and `os.Root` re-resolves each directory from the
containment root one component at a time, making a directory chain quadratic
in its depth (depth-1500 did not finish in two minutes). Directories are now
counted against the same cap and descent is capped at `maxWalkDepth`. The
general rule: any new whole-tree traversal states which of {entries, depth,
aggregate bytes} bounds it, because a per-item cap and a count cap multiply
and their product is not a bound.
