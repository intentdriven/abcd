---
schema_version: 1
id: "iss-2609291233390280"
slug: "dot-glob-bracket-expressions-read-as"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix6-guardGlob"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "A dot-led segment holding a bracket expression the glob compare cannot decide reads as able to match .., so ~/.[[:punct:]]/*, ~/.[].]/*, ~/.[--.]/*, ~/.[\\!.]/* and the rest of the verify's spellings block, bare and through sh -c and bash -c; ~/.[a-z]/* blocks as a stated over-block."
impact: fix
resolved_by:
  commit: "9a430d8b9680a2f8928ab93bd109778ccc606bf8"
---

The guard reads a dot-led glob segment with a bracket expression as unable to match .., so rm -rf ~/.[[:punct:]]/*, ~/.[[:print:]]/*, ~/.[![:alnum:]]/*, ~/.[[=.=]]/*, ~/.[].]/*, ~/.[!]]/*, ~/.[--.]/*, ~/.[\!.]/*, $HOME/../.[[:punct:]]/*, /.[[:punct:]]/* and ~/.[[:punct:]]/** allow, bare and through sh -c or bash -c, while bash 3.2 and /bin/sh expand each to the entries of the directory holding the home (or the root). dotGlob decides through path.Match, which has no POSIX, equivalence or collating classes, refuses a set whose first member is ] or -, and sees \! after the tokenizer has removed the backslash, so an escaped ! reads as a negation. Pre-existing at the round's base 4e8cbd387; found by verify-fix6-guardGlob item 1.

## Grounds

- pursued: every spelling in verify-fix6-guardGlob item 1 blocks via rm-rf-root-or-home (TestDotGlobBracketExpressionsReadAsParent); a dot-led bracket segment that bash 3.2 or /bin/sh expands to .. and the guard allows would show it wrong
