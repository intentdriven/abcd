---
schema_version: 1
id: "iss-2609281017573862"
slug: "the-abcd-path-entry-and-cache-attestation-readers-do-not"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: review-drainH"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil"
resolution: "Every reader of a file abcd trusts in ~/.abcd reads through fsutil.ReadHomeDeclaration, which refuses a file behind a symlinked ~/.abcd (fsutil.HomeScopeLink) and reads an empty symlinked ~/.abcd as absent; path-entry and cache-attestation are refused through ahoy.homeScope, the hook shims test the link before reading path-entry, and hooks/bootstrap.sh refuses to write either record through it."
impact: fix
resolved_by:
  commit: "734b26af4"
---

The ~/.abcd/path-entry and cache-attestation readers do not refuse a symlinked ~/.abcd, unlike the rules loader: fsutil.ReadDeclaration lstats the FILE (regular, this uid, not group- or other-writable, O_NOFOLLOW re-open) but follows a symlinked parent directory, and the hook shims' find -maxdepth 0 -type f -user ... ! -perm -0020 ! -perm -0002 check likewise follows the parent. The two readers agree with each other, but AGENTS.md states the rules.json rule as refused behind a symlinked ~/.abcd, so a dotfiles-symlinked ~/.abcd hosts a path-entry record that vouches for which binary the hooks run while the rules loader in the same home would refuse its rules.json. Found by review-drainH (LOW, pre-existing, shared with cache-attestation); related to iss-2609260958587561 (the credential store's copy of the same gap) but not covered by it.

## Grounds

- pursued: a well-formed, owned, owner-only path-entry or cache-attestation behind a symlinked ~/.abcd vouches for nothing in ahoy, the hook shims run no PATH abcd on it, and the bootstrap writes nothing behind the link (TestHomeScopedRecordsRefuseASymlinkedAbcdHome, TestBinaryHooksRefuseAPathBinaryVouchedForBehindASymlinkedAbcdHome, TestBootstrapRefusesASymlinkedAbcdHome); a reader that still followed the link, or a bare fsutil.ReadDeclaration call outside fsutil, would show it wrong (TestHomeDeclarationsReadThroughReadHomeDeclaration).
