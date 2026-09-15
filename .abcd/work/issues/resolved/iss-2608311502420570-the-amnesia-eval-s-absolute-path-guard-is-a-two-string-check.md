---
schema_version: 1
id: "iss-2608311502420570"
slug: "the-amnesia-eval-s-absolute-path-guard-is-a-two-string-check"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-187 fidelity audit rcp-d3041aa2b510"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_determinism_test.go"
resolution: "The amnesia eval's absolute-path guard is a path detector on two mechanisms: every ancestor of the fixture root and HOME up to the process temporary directory, which is where the shared parent lives, and a shape match for any other absolute path. Each half is falsified separately, and a planted shared parent that the two-string check passed is now caught."
impact: internal
---

The amnesia eval's absolute-path guard is a two-string check rather than a path detector, so it misses the leak class it was added to close. It looks for the two fixture roots by name only. Both roots are created under one temporary parent, and planting that shared parent string in the bundle leaves the whole lane green: the byte comparison cannot see it, because both runs carry the same string and therefore still agree, and the subtest is not looking for it. The two-path design exists precisely because a run-to-run comparison in one directory cannot see an absolute path embedded in the output, and this guard reintroduces that blindness one level up. It is also the repository's own privacy rule at stake, not only determinism: no absolute local path may enter an artefact.

## Grounds

- pursued: a leak class one level above a two-string list is closed by a detector rather than by a longer list, because the next level up would need another entry; wrong if a legitimate assembly reports a path it does not carry
