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
---

The shell guard reads an unquoted fixed here-document output as its split words only where the word is wholly that substitution. Text glued to it (the substitution followed by x) or two such substitutions glued together fall to the unknown reading and reach ALLOW in command position, while bash runs the joined words: the hazard with x appended to its last word, or the hazard assembled from the two documents (review7-guard finding 3, verified with a neutral word). The guard page and the brief say the words are read wherever the substitution stands, which is false for a glued word.
