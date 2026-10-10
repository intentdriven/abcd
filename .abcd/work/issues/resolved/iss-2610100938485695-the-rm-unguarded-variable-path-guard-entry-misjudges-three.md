---
schema_version: 1
id: "iss-2610100938485695"
slug: "the-rm-unguarded-variable-path-guard-entry-misjudges-three"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "Fable review of the rm-unguarded-variable-path entry, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/varpath.go"
remedy: "Carry the non-empty guarantee of :? / :- / := through the named re-read of a double-quoted shell string, read nested defaults recursively, and treat a variable followed by a substitution before the slash as emptyable; one fixture per shape, watched fail first."
resolution: "The rm-unguarded-variable-path entry now reads all three shapes: a double-quoted shell string's re-read takes its lead from the string written with each guarded value as its guard, so sh -c \"rm -rf ${X:?}/y\" is allowed; a guard nested in a default is read (and a name the expansion also writes unguarded, such as ${VAR:-${VAR}}, is not guarded); and a variable followed by a command substitution before the slash refuses. Arithmetic expansion stays allowed, since it always prints a number."
impact: fix
---

The rm-unguarded-variable-path guard entry misjudges three edge shapes found in its review. (1) Inside a double-quoted shell string the successor's own rewrite is refused: sh -c "rm -rf ${X:?}/y" blocks, because the re-read string spells ${X:?} as ${X} (payload.go spellParameterAt), so the refusal tells the agent to do what it did; single quotes avoid it. (2) A nested guard is not read: "${VAR:-${OTHER:?}}"/y and "${VAR:-${OTHER:-/tmp}}"/y block although both are guarded (varpath.go stripEmptyableRefs strips the inner reference). (3) A site followed by a substitution is not followed: rm -rf $VAR$(true)/x is allowed (varpath.go).
