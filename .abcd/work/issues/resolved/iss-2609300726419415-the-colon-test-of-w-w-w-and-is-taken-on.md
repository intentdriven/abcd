---
schema_version: 1
id: "iss-2609300726419415"
slug: "the-colon-test-of-w-w-w-and-is-taken-on"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Apply the colon forms' set reading (no empty value) only to a single positional or special parameter; for @ and * keep the value with its empty text, so ${@:-x}/ blocks and ${1:-dist}/ stays allowed. Grounds: the bash manual, Shell Parameter Expansion, and the four shells' printf output with set -- \"\" \"\"."
resolution: "The colon forms of @ and * keep the empty value, since their colon test is on the parameter count; the set reading applies to a single positional or special parameter only."
impact: fix
resolved_by:
  commit: "ff4016a84"
---

The colon test of ${@:-w}, ${*:-w}, ${@:=w} and ${@:?} is taken on the parameter count, not on a joined value, but the guard read them as a single empty parameter does (the empty value never printed), a regression from iss-2609300651127327. With set -- "" "" (a script called as clean.sh "" "") bash 3.2, /bin/sh, dash and bash 5.3 hand rm the home for rm -rf $HOME${@:-x} and $HOME${*:?}, and / or /* for rm -rf ${@:-x}/*, ${*:-x}/ and ${@:?}/; all were allowed.

## Grounds

- pursued: ${@:-x}/, ${@:-x}/*, ${*:-x}/, ${@:?}/, ${*:=x}/, $HOME${@:-x} and $HOME${*:?} block bare, in bash -c and in sh -c, while ${1:-dist}/ stays allowed; a shell that printed dist/ for ${@:-x}/ after set -- "" "" would show it wrong
