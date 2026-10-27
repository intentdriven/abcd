---
schema_version: 1
id: "iss-2610241722201794"
slug: "the-product-thinker-asked-for-every"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-24, planning for a future Windows build"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts"
remedy: "Review every shell script and classify each: (a) must stay a shell because it runs before the binary exists, such as hooks/bootstrap.sh and the hook shims, then cut it to a minimal launcher with a Windows sibling; (b) belongs in the Go binary as an abcd verb or a hidden gate, such as the check-* gates and their cases, preflight-receipt, pinned-toolchain, pr-keep-current and dependency-reauthor; (c) runs only on CI's Linux runners and may stay. Record the classification as a decision, move the (b) scripts into Go behind their existing tests, and add a Windows lane to CI that runs the moved verbs."
---

The product thinker asked for every shell script in abcd to be reviewed for which should be built into the Go binary, so abcd can run on other systems, Windows among them. Today abcd's behaviour leans on POSIX shell in about fifty places: 37 tracked .sh files (14 under scripts/cloud, 13 gate and helper scripts under scripts/ such as check-attribution, check-issue-resolution, check-reviews, check-wip-source, check-vcs-stamp, preflight-receipt, pinned-toolchain, pr-keep-current and dependency-reauthor, and 10 plugin hooks under hooks/, bootstrap.sh among them); five git hooks in .githooks/ and their four templates in internal/core/ahoy/defaults/, all POSIX sh; the install script template under site-src/; and the Makefile that runs preflight. Each needs a POSIX shell, and many need GNU or BSD userland tools, so none of them runs on a stock Windows machine, and a Windows build of the binary alone would leave the gates, hooks and preflight behind.
