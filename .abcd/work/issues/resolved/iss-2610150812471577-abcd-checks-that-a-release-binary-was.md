---
schema_version: 1
id: "iss-2610150812471577"
slug: "abcd-checks-that-a-release-binary-was"
severity: "major"
category: "security"
source: "review-followup"
found_during: "abcdesign publishing workflow security review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/site.yml"
remedy: "Pin every attestation check to release.yml@refs/heads/main with --source-ref and --source-digest of the tag's commit; turn on immutable releases; give the release environment a main-and-v*-tags policy; restrict v* tag creation to admins. The repository settings are the product thinker's call, made with the release-cutting session"
resolution: "Every provenance check now requires release.yml on main, the released commit and a GitHub-hosted runner, and update/provenance.go accepts main only (#1023); releases are drafted, every asset is verified by id, and only then published; immutable releases are on and the release environment accepts main and v* tags only. Shipped in v0.15.0."
impact: fix
shipped_in: v0.15.0
resolved_by:
  commit: "6d2bd64b8"
---

abcd checked that a release binary was built by its release workflow, but most of those checks named only the workflow file and never the branch or tag it ran from, so anyone with write access who pushed a branch or a v-tag carrying an edited release.yml could sign a binary the check accepted (`gh attestation verify --signer-workflow` matches the workflow path and ignores the ref after `@`). The checks that named the path only were `.github/workflows/release.yml` (the receipts check, the linux binary and the plugin zip), `.github/workflows/site.yml`, and the release workflow template laid into managed repositories; the bootstrap installer, `install.sh`, the install one-liners and the printed by-hand commands added the tag's commit but not the ref; and `internal/core/update/provenance.go` pinned the ref but also accepted any `refs/tags/v*`. Underneath, abcd's releases were mutable, so a release's binary and `checksums.txt` could be replaced together, and the release environment had no branch policy.

Advisory: [GHSA-cq4f-8v48-9h97](https://github.com/intentdriven/abcd/security/advisories/GHSA-cq4f-8v48-9h97), published 2026-10-16 with v0.15.0 as the patched version. The record was held out of the public tree until then.

The product thinker ruled on 2026-10-15 that the weakness is fixed before the release after v0.14.2, approved the repository settings, and, when restricting v-tag creation proved impossible without blocking the release workflow (it pushes the tag with the workflow token, which a repository ruleset cannot exempt), ruled that tag creation stays open: a hand-pushed tag cannot start a release run, and every check requires a binary built from main.
