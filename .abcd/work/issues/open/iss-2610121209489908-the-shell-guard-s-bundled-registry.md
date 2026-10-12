---
schema_version: 1
id: "iss-2610121209489908"
slug: "the-shell-guard-s-bundled-registry"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "security review of fix/hooks-run-head-in-dogfood-roots, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "survey which abcd readers trust refs/remotes/origin as the reviewed tip; for each that does, either verify against a declared remote as the dogfood build now does, or add guard registry entries that refuse a local write under refs/remotes/<remote>/ from a non-remote source and an edit of remote.<name>.* config"
---

The shell guard's bundled registry (internal/core/guard/defaults/guard.json) refuses none of the ordinary commands that move a remote-tracking ref by hand: git fetch . +refs/heads/<x>:refs/remotes/origin/<y>, git remote set-head, git remote set-url, an edit of remote.origin.fetch, or git update-ref and git symbolic-ref under refs/remotes/. The dogfood build's security review showed four of these moving refs/remotes/origin/main to an agent's commit with the guard answering exit 0; the dogfood fix makes the build source a remote the person declares, so that route closes without them, but any other reader that trusts refs/remotes/origin as the reviewed tip is still exposed to them.
