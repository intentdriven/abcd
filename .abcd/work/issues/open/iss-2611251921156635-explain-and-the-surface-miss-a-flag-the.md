---
schema_version: 1
id: "iss-2611251921156635"
slug: "explain-and-the-surface-miss-a-flag-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/release/surface.json"
remedy: "Give a flag enforced inside the run the same mark the explain output and the surface snapshot read, set in one place beside the refusal, and let the release check compare on it. Grounds: the refusal and the snapshot describe one flag and disagree."
---

abcd explain capture gives --remedy no mark of being required, plain or with --json, and .abcd/development/release/surface.json records it as not required, while abcd capture without it is refused with exit 2. The requirement is made inside the command, so the surface snapshot, and the release check that names a newly required flag as a break, cannot see it. Tested on 2026-11-25; the code is the same on main.
