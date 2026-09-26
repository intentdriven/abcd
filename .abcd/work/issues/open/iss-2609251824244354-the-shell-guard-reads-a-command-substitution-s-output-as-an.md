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
deferral_reason: "fix round 2 of lane guard (run A 2026-09-25) was scoped by its brief to command substitutions, fix round 3 resolved the command-position half, and fix round 4 the ${…} that carries a substitution; reading $VAR, a ${…} holding no substitution and $@ as unknown words touches every variable in the everyday corpus (git push origin \"$branch\", gh api paths, eval \"$X\") and needs its own tokenizer pass for ${…} bodies and its own false-positive sweep. The unknown-word primitive (unknown.go), read by every word reader, is the seam it lands on."
---

The shell guard reads a command substitution's output as an unknown word, but not a parameter expansion's. A dash glued to a variable (git push --$X origin main, rm -$F after a cd) is read as the literal text --$X, which names no flag, so every blocker allows it while its --$(echo x) twin blocks; and ${GIT:-git} standing in command position is compared as text. bash builds the hazard from either. Found while closing review2-guard (its finding 1 names --$X as pre-existing). The record named a second half, a substitution standing in command position, which fix round 3 of lane guard resolved (a name a substitution prints is every program its known tail allows); this record is the parameter-expansion half alone. Fix round 4 made a ${…} that carries a command substitution (${X:-$(…)}, --${X:-$(…)}) unknown from its ${ on (iss-2609252120211621). What remains is exactly a parameter expansion with no substitution in it: $X, ${X}, ${X:-word} and the other operators with literal words, and $@ / $*, standing as the command's program name, as a flag or glued to one, as an operand an entry constrains, or inside a payload the guard reads.
