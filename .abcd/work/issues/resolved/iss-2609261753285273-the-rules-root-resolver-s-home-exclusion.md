---
schema_version: 1
id: "iss-2609261753285273"
slug: "the-rules-root-resolver-s-home-exclusion"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/root.go"
resolution: "Both home sites in rules.Resolve ask one helper, homeMatcher, which compares the home and the candidate by file identity (os.Stat plus os.SameFile), so a trailing-slash, symlinked or case-variant HOME is still the home."
impact: fix
resolved_by:
  commit: "c742b4f6"
---

The rules root resolver's home exclusion compares path strings (root.go dir != home in the walk, top == home after it), so a HOME spelled as a case variant of the on-disk path (the home spelled with a lower-case first component where the case-insensitive APFS volume stores it capitalised) is not recognised: filepath.EvalSymlinks keeps the caller's case while git reports the on-disk case, and the version-controlled home is adopted as the repo root, its .abcd read a second time as the repo layer. Compare by file identity (os.Stat plus os.SameFile) through one helper at both sites.

## Grounds

- pursued: a HOME naming the home directory at any spelling is never adopted as the repo root, pinned by TestResolveRootNeverAdoptsTheHomeAtAnySpelling at both sites; a case-variant or symlinked HOME resolving to the home as Root on a case-insensitive volume would show it wrong
