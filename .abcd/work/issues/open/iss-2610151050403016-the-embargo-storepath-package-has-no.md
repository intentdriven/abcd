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
---

The embargo store's path moved into a leaf package, internal/core/embargo/storepath, to break an import cycle when the embargo branch merged main (2026-10-15, PR #1015). The package has no test of its own; its rule is exercised only through the embargo and peers tests.
