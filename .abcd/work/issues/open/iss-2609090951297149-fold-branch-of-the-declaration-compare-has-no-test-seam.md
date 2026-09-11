---
schema_version: 1
id: "iss-2609090951297149"
slug: "fold-branch-of-the-declaration-compare-has-no-test-seam"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/root.go"
---

Both user-scope declaration readers call the exported case-folding predicate directly when comparing a declared path against the target: the rules resolver and the transcript-store locator each ask whether the filesystem folds case and then compare under that answer. Two sibling packages hold the same predicate in a package-level variable for the stated reason that substituting it is the only way a test can exercise the branch, and the path utilities package keeps one for its own redactor. Neither of these two does, so on a case-sensitive host the fold-equality branch of the declaration compare is unreachable by any test, and what it does with a case-variant spelling of a declared root is pinned nowhere. Verified by grepping the seam: it exists in the path utilities, the ahoy package and the update package, and not in rules or history. It matters because the declaration is the opt-in that re-admits a foreign-owned checkout, so a fold bug there either silently admits a root the caller never declared or silently refuses one they did, and the case-sensitive CI leg would show neither. Fix direction: hold the predicate in a package variable in both packages, exactly as the two siblings already do, and add the case-variant declaration case each seam makes possible. Detector: with the fold predicate forced on, a declaration written in a case variant of the target must be honoured, and with it forced off it must not.
