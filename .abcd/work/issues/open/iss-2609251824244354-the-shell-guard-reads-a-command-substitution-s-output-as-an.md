---
schema_version: 1
id: "iss-2609251824244354"
slug: "the-shell-guard-reads-a-command-substitution-s-output-as-an"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
deferred_after: "v0.10.0"
deferral_reason: "fix round 2 of lane guard (run A 2026-09-25) was scoped by its brief to command substitutions; reading $VAR, ${…} and $@ as unknown words touches every variable in the everyday corpus (git push origin \"$branch\", gh api paths, eval \"$X\") and needs its own tokenizer pass for ${…} bodies and its own false-positive sweep, and an unknown command name needs a ruling on whether `\"$(which git)\" push` style launchers block or warn. The unknown-word primitive (unknown.go) is the seam both land on."
---

The shell guard reads a command substitution's output as an unknown word, but not a parameter expansion's, nor a substitution standing in command position. A dash glued to a variable (git push --$X origin main, rm -$F after a cd) is read as the literal text --$X, which names no flag, so every blocker allows it while its --$(echo x) twin blocks; and "$(which git)" push followed by a blocked flag allows, because the unknown command name is compared as text. bash builds the hazard from either. Found while closing review2-guard (its finding 1 names --$X as pre-existing).
