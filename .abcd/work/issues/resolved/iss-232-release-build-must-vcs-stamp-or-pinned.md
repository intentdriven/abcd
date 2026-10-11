---
schema_version: 1
id: "iss-232"
slug: "release-build-must-vcs-stamp-or-pinned"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-111 independent ruthless review (2026-08-16)"
found_at: ".github/workflows/release.yml + internal/core/vintage"
resolution: "scripts/check-vcs-stamp.sh asserts vcs.revision present AND vcs.modified false over every bin/abcd-* binary; release.yml and its scaffold template run it between build and checksums, aborting the release before publish"
impact: internal
resolved_by:
  commit: "f6bef89"
---

itd-111 latent coupling: the ahoy install staleness refusal keys on the binary's vcs.revision (via vintage current-from-BuildInfo). Release binaries are VCS-stamped today (release.yml runs make build VERSION=<tag> from a git checkout, -buildvcs=auto default), so pinned users have a determinable vintage and proceed. But if a future release build ever sets -buildvcs=false or builds from a .git-less source archive, Known flips false for every pinned user and ahoy install begins REFUSING until they pass --allow-stale-binary. Add a guard/test on the release build config asserting the shipped binary carries vcs.revision, so this coupling cannot silently break. Non-blocking; surfaced by the independent ruthless review of feat/itd-111-staleness. NOTE: captured in the iss-200 design worktree to avoid an allocator collision (iss-231 taken there); it is an itd-111/release-build follow-up, not iss-200 work.