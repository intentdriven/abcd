---
schema_version: 1
id: "iss-2609290745243990"
slug: "rm-rf-root-or-home-reads-a-parent-segment-only-at-the-root"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "foldParents (match.go) folds each .. into the segment before it where the target begins at the root or the home, lexically: /tmp/../* is /*, and a .. past the home climbs to a directory holding the home, so ~/../* and ~/.. read as ~ and ~/../*/* as ~/*. TestParentSegmentsThatReachTheRootOrTheHome pins 32 blocks (96 spellings, all red at 9eb42ff56) and 11 allow/warn siblings. Residuals, named in 17-guard.md and not soundly decidable from the text: a .. after a symlink is read lexically; nothing is folded across a segment holding a variable (/tmp/$X/../../* is the root with X unset, the unset-variable class the chapter already names); a glob other than * past the home (~/../?*) stays, as /?* does."
impact: fix
resolved_by:
  commit: "c811c7d7c"
---

rm-rf-root-or-home reads a parent segment only at the root: rm -rf ~/../* deletes the home, and rm -rf ~/../../*, $HOME/../../*, /tmp/../* and /etc/../* delete the root, bare and in sh -c, yet each allows, because cleanSeparators takes out only a leading /../ and never folds a .. after a named directory, the home or a tilde. A root or home delete is allowed.

## Grounds

- pursued: every operand beginning at /, ~, $HOME or ${HOME} whose lexical reading is the root, or a directory holding the home globbed or deleted whole, blocks bare and in sh -c and bash -c; a lexical root or home spelling of that shape that still allows would show it wrong
