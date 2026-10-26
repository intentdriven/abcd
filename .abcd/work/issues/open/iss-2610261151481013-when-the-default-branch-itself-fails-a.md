---
schema_version: 1
id: "iss-2610261151481013"
slug: "when-the-default-branch-itself-fails-a"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "CI-escape analysis, 2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "In the preflight vulncheck step (iss-2610261151381604), when govulncheck reports a finding and the branch changes neither go.mod nor go.sum against its merge base with origin/main, say that the finding most likely stands on origin/main too, name each fix govulncheck names (the toolchain or module version it is fixed in), and stop before the test stages; a branch that does change them is told the finding may be its own"
blocked_by: [iss-2610261151381604]
---

When the default branch itself fails a hosted check, every branch pushed meanwhile pays a hosted run that is certain to fail, and nothing local says so before the push. From 2026-10-21 to 2026-10-22 a newly published Go standard-library vulnerability failed govulncheck on the nightly run and then on every pull request and queue entry, 10 runs, for about a day and a half, until the toolchain bump landed. A branch that changed neither go.mod nor go.sum could not have introduced it, and its author could have been told to land the bump first.
