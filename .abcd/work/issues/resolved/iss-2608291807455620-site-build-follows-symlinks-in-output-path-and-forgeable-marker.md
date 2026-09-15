---
schema_version: 1
id: "iss-2608291807455620"
slug: "site-build-follows-symlinks-in-output-path-and-forgeable-marker"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "v0.6.9-security-pass"
found_at: "internal/core/site/build.go"
resolution: "site build and site check pass the output path through one gate that refuses a symlink at the leaf or at any ancestor inside the repository, the repository root and any .git holder, and the purge additionally requires a marker naming this repository's root commit and a directory git tracks nothing in; a repository that commits its built site directory is refused by abcd site build and must be pointed at an untracked output directory"
impact: fix
---

GHSA-fpf2-pg82-72rj: abcd site build and site check follow symlinks in the output path, and the .abcd-site-build marker is name-only. inspectOutDir uses os.ReadDir/os.Stat (follows a symlinked leaf), purgeOutDir runs os.RemoveAll through the same path, and MkdirAll plus os.OpenRoot open a symlink's target as the containment root; no Lstat/EvalSymlinks guard exists, and an ancestor symlink is missed even by a leaf-only Lstat. A committed marker forges ours classification so a later --out purges the directory's other entries (with --out . that includes .git). CWE-59.
