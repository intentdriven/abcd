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
---

The ~/.abcd/path-entry and cache-attestation readers do not refuse a symlinked ~/.abcd, unlike the rules loader: fsutil.ReadDeclaration lstats the FILE (regular, this uid, not group- or other-writable, O_NOFOLLOW re-open) but follows a symlinked parent directory, and the hook shims' find -maxdepth 0 -type f -user ... ! -perm -0020 ! -perm -0002 check likewise follows the parent. The two readers agree with each other, but AGENTS.md states the rules.json rule as refused behind a symlinked ~/.abcd, so a dotfiles-symlinked ~/.abcd hosts a path-entry record that vouches for which binary the hooks run while the rules loader in the same home would refuse its rules.json. Found by review-drainH (LOW, pre-existing, shared with cache-attestation); related to iss-2609260958587561 (the credential store's copy of the same gap) but not covered by it.
