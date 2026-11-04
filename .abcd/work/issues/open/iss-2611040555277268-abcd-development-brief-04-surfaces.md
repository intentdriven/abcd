---
schema_version: 1
id: "iss-2611040555277268"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "State that the final component must be a real directory wherever it is."
---

.abcd/development/brief/04-surfaces/11-history.md:147-149 says --out is refused through a symlink at any level inside a checkout and that outside every checkout the path is the operator's own. A symlink as the last component of --out is refused outside a checkout too, exit 1, reporting that --out is not an existing directory (internal/surface/cli/history_reconstruct.go:118, fsutil.IsRealDir uses Lstat). A symlinked parent outside a checkout is followed, as the chapter says.
