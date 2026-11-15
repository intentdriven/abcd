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
resolution: "The history chapter of the brief now says that outside every checkout only the levels above the last are the operator's own, and that the last component of history reconstruct's --out must itself be a real directory wherever it is; a symlink there is refused outside a checkout too."
impact: internal
resolved_by:
  commit: "dd666656419cb199e66cd34af3fa40c1f90e94b3"
---

.abcd/development/brief/04-surfaces/11-history.md:147-149 says --out is refused through a symlink at any level inside a checkout and that outside every checkout the path is the operator's own. A symlink as the last component of --out is refused outside a checkout too, exit 1, reporting that --out is not an existing directory (internal/surface/cli/history_reconstruct.go:118, fsutil.IsRealDir uses Lstat). A symlinked parent outside a checkout is followed, as the chapter says.

## Grounds

- pursued: the chapter matches the verb, which refuses a symlinked last component of --out outside a checkout and follows a symlinked parent there; TestReconstructOutLeafMustBeARealDirectoryOutsideACheckout failing on either behaviour or on the chapter's sentence would show it wrong
