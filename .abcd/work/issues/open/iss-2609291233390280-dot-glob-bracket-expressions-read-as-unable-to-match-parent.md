---
schema_version: 1
id: "iss-2609291233390280"
slug: "dot-glob-bracket-expressions-read-as-unable-to-match-parent"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix6-guardGlob"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

The guard reads a dot-led glob segment with a bracket expression as unable to match .., so rm -rf ~/.[[:punct:]]/*, ~/.[[:print:]]/*, ~/.[![:alnum:]]/*, ~/.[[=.=]]/*, ~/.[].]/*, ~/.[!]]/*, ~/.[--.]/*, ~/.[\!.]/*, $HOME/../.[[:punct:]]/*, /.[[:punct:]]/* and ~/.[[:punct:]]/** allow, bare and through sh -c or bash -c, while bash 3.2 and /bin/sh expand each to the entries of the directory holding the home (or the root). dotGlob decides through path.Match, which has no POSIX, equivalence or collating classes, refuses a set whose first member is ] or -, and sees \! after the tokenizer has removed the backslash, so an escaped ! reads as a negation. Pre-existing at the round's base 4e8cbd387; found by verify-fix6-guardGlob item 1.
