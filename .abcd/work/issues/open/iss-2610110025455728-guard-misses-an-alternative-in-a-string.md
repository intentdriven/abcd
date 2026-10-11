---
schema_version: 1
id: "iss-2610110025455728"
slug: "guard-misses-an-alternative-in-a-string"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of the rm-unguarded-variable-path edge fix (fix/guard-rm-varpath-edges)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "When a payload site's text is empty and no text it prints names a variable (an alternative such as ${X:+x}), write the empty text in the lead spelling as a reference that can be empty, so the string's re-read leads with a variable as the direct form does; one fixture, watched fail first."
---

The rm-unguarded-variable-path guard entry allows an alternative inside a double-quoted shell string: sh -c "rm -rf ${X:+x}/y" is allowed, although the enclosing shell expands ${X:+x} to nothing when X is unset and the string's shell then deletes /y. The direct form, rm -rf ${X:+x}/y, refuses. The cause is the payload re-read: the alternative's texts are the empty text and x, neither names a variable, so the string is written out as rm -rf /y and rm -rf x/y and neither reading leads with a variable (payload.go namedPayloads, varpath.go leadSpelling). It has been allowed since the entry landed in #889. Separately, the edge fix refuses sh -c "rm -rf ${X:-\\ }/y", whose escaped blank the string's shell keeps as text; this is a deliberate over-refusal (varpath.go emptiedInside) and main refused it too.
