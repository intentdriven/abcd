---
schema_version: 1
id: "iss-2609260115380561"
slug: "the-shell-guard-misreads-a-fixed-output-glued-to-other-text"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A word holding a fixed here-document output is read as bash builds it: an unquoted output split on the default IFS with its first and last words joined to the text beside it, a quoted one joined to the text as one word, so a glued or doubled fixed output in command position, behind a wrapper or as a payload blocks on the entry it spells; the page, the brief and the help say what is read."
impact: fix
resolved_by:
  commit: "d1703108"
---

The shell guard reads an unquoted fixed here-document output as its split words only where the word is wholly that substitution. Text glued to it (the substitution followed by x) or two such substitutions glued together fall to the unknown reading and reach ALLOW in command position, while bash runs the joined words: the hazard with x appended to its last word, or the hazard assembled from the two documents (review7-guard finding 3, verified with a neutral word). The guard page and the brief say the words are read wherever the substitution stands, which is false for a glued word.

## Grounds

- pursued: the review's glued and doubled shapes block, and the shapes where the joined word is not the hazard still allow; a glued fixed output bash runs that still allows would show it wrong
