---
schema_version: 1
id: "iss-2609020352438590"
slug: "both-containment-checks-that-decide-whether-abcd-runs-a-fore"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
---

Both containment checks that decide whether abcd runs a foreign binary or reads a foreign cache are weaker than the prose around them claims, in two shared ways. (1) The hook shims' PATH rung (hooks/hooks.json, all four PATH-resolving events) compares the candidate binary's directory against the shim's own `pwd -P`, not against the root of the repository the session is working on, so `<repo>/vendor/bin/abcd` is refused from `<repo>` and accepted from `<repo>/sub` — the same hostile clone, a different working directory. (2) Both that rung and dataDirHazard in internal/core/ahoy/data_dir.go judge only the containing directory's mode, so a world-writable file (0777) inside an ordinary 0755 directory passes every check; on a system where the directory's owner is not the only writer of its contents, the binary that is executed is still anyone's to replace. A fix must establish that containment is measured against the repository root the shim is protecting (git rev-parse --show-toplevel, or the harness's project dir), and that the trust test covers the artefact's own mode and ownership, not just its parent's.

(3) Related, found while tightening the rung's own tests: which of the rung's
refusals fires for a RELATIVE PATH element is decided by the shell, not by the
shim. dash — `/bin/sh` on the Linux leg — answers `command -v abcd` with
`./abcd` verbatim, so the rung refuses it for not being absolute; bash in sh
mode — `/bin/sh` on macOS — resolves a relative PATH element against `$PWD`
first, so the rung sees an absolute path and refuses it as inside the working
tree. Both refuse, so nothing is exposed here. But the two shells diverge in a
case that is NOT covered: a relative PATH element pointing OUT of the tree
(`PATH=../elsewhere` from `<repo>/sub`). dash refuses it as non-absolute; bash
absolutises it, finds the physical directory outside the working directory, and
accepts it. The same fix — measuring containment against the repository root,
and deciding what a relative PATH element means before the shell does — closes
this with the other two.

## Two further shapes from the second security review (2026-09-02)

- The containment and world-writable tests judge the directory the PATH entry
  lives in, not the binary it resolves to: a symlink in an ordinary directory
  (`~/.local/bin/abcd -> <repo>/inside/abcd`) launders both refusals, because
  `dd=${c%/*}` names the link's directory and `cd -P` resolves that, never the
  target. docs/how-to/install.md states the accepted shape in terms of the
  binary actually executed, which is false for this case.
- dataDirHazard checks the data directory and its cache only: a world-writable
  non-sticky ANCESTOR (rename-and-substitute) and a world-writable artefact file
  inside a 0755 cache both pass.

