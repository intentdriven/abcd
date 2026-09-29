---
schema_version: 1
id: "iss-2609290426544292"
slug: "rm-rf-root-or-home-reads-a-default-expansion-by-its-variable"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Make segment.spelled a set of the texts a word can print (varSite.texts, spellWritten), read a default's and an alternative's word through it (spellWord, the alternative also printing nothing), a substring's leading /, and a * replacement's string, pair every text through spellPayload, and bound depth and size so past either the word refuses, test first."
deferred_after: v0.11.1
deferral_reason: "Reading a default's word needs a written spelling that holds more than one text (the variable's value or the default's word), which changes segment.spelled from one string per word to a set and the payload pairing that copies it (spellPayload); owed: that representation, then the default word, deep alternatives and a substring's root read through it, test first."
---

rm-rf-root-or-home reads a default expansion by its variable only: rm -rf ${DIR:-$HOME} and rm -rf ${DIR:-/} delete the home or the root when DIR is unset and allow, because a word's written spelling holds one text and the default's own word is the other value it can print. An alternative nested more than three deep (${X:+${X:+${X:+${X:+$HOME}}}}) and ${PWD:0:1}, which prints the root and warns as $PWD, are the same class. Named in 17-guard.md's residuals.

## Deferral 2026-09-29

Deferred past v0.11.1: Reading a default's word needs a written spelling that holds more than one text (the variable's value or the default's word), which changes segment.spelled from one string per word to a set and the payload pairing that copies it (spellPayload); owed: that representation, then the default word, deep alternatives and a substring's root read through it, test first.

## Evidence 2026-09-29: a default after a subscript

The class includes a default the bash 3.2 of macOS reads at the first operator after a subscript's `]`. With X unset, bash 3.2 and /bin/sh print the word for `${X[0]]-$HOME}`, `${X[0]]:-$HOME}`, `${X[0]]=$HOME}`, `${X[0]]:=$HOME}` and `${X[0]]x-$HOME}` (the home), and for `${X[0]]-/}` (the root); bash 5 refuses each as a bad substitution. With X set each prints X's value. The guard reads the subscript's operator (unknown.go subscriptOperators) and spells a `-` or `=` there as the variable, as it spells `${X:-$HOME}`, so each allows. The pin `${X[0]]-$HOME}` in homeresiduals_test.go is this residual, not a claim that the form stays off the home. The same owed representation, a spelling that holds both texts, reads them.
