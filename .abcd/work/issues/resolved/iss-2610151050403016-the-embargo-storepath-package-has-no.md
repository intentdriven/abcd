---
schema_version: 1
id: "iss-2610151050403016"
slug: "the-embargo-storepath-package-has-no"
severity: "nitpick"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/peers"
remedy: "Add a storepath test covering a repository with a root commit, one without, and a SHA-256 repository, so the path rule is pinned where it now lives rather than only through embargo's and peers' tests."
resolution: "The root-commit and no-commit asks were already covered by existing storepath tests: TestLocateNoStoreIsUnderTheOldestRoot (a repository with a root commit keys the store under that root) and the no commit subtest of TestLocateRefusesRootsItCannotRead (a repository with no commit is ErrNoRootCommit). This change adds the SHA-256 case, TestLocateKeysASHA256RepositoryOnItsFullRootName: a repository made with --object-format=sha256 has its store placed under the root commit's full 64-digit name, and a store registered there is found."
impact: internal
resolved_by:
  commit: "32a0607ff13e00c659bb4f108a6070acafc058f1"
---

The embargo store's path moved into a leaf package, internal/core/embargo/storepath, to break an import cycle when the embargo branch merged main (2026-10-15, PR #1015). The package has no test of its own; its rule is exercised only through the embargo and peers tests.

## Grounds

- pursued: Locate keys a SHA-256 repository's store on the full 64-digit root commit name; a 40-digit (SHA-1-length) expectation fails the test, as detector-red.log shows, and any storepath change that cut or mis-read a SHA-256 root would fail it the same way.
